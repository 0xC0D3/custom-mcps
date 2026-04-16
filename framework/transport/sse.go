package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/0xC0D3/custom-mcps/framework/auth"
)

// SSETransport implements the legacy SSE-based MCP transport with separate
// endpoints for the SSE stream (GET) and message submission (POST).
type SSETransport struct {
	addr        string
	sseEndpoint string
	msgEndpoint string
	httpServer  *http.Server
	sessions    *sessionManager
	auth        auth.Authenticator
	httpMws     []HTTPMiddleware
	logger      *slog.Logger
	tlsCert     string
	tlsKey      string
	healthPath  string
}

// SSEOption configures an SSETransport.
type SSEOption func(*SSETransport)

// WithSSEAddr sets the listen address.
func WithSSEAddr(addr string) SSEOption {
	return func(t *SSETransport) { t.addr = addr }
}

// WithSSEEndpoint sets the SSE stream endpoint path.
func WithSSEEndpoint(endpoint string) SSEOption {
	return func(t *SSETransport) { t.sseEndpoint = endpoint }
}

// WithSSEMsgEndpoint sets the message submission endpoint path.
func WithSSEMsgEndpoint(endpoint string) SSEOption {
	return func(t *SSETransport) { t.msgEndpoint = endpoint }
}

// WithSSEAuth sets the authenticator.
func WithSSEAuth(a auth.Authenticator) SSEOption {
	return func(t *SSETransport) { t.auth = a }
}

// WithSSEHTTPMiddleware adds HTTP middlewares.
func WithSSEHTTPMiddleware(mws ...HTTPMiddleware) SSEOption {
	return func(t *SSETransport) { t.httpMws = append(t.httpMws, mws...) }
}

// WithSSELogger sets the logger.
func WithSSELogger(l *slog.Logger) SSEOption {
	return func(t *SSETransport) { t.logger = l }
}

// WithSSETLS enables TLS with the given certificate and key files.
func WithSSETLS(certFile, keyFile string) SSEOption {
	return func(t *SSETransport) { t.tlsCert = certFile; t.tlsKey = keyFile }
}

// WithSSEHealthPath sets the health endpoint path. Empty string disables it.
func WithSSEHealthPath(path string) SSEOption {
	return func(t *SSETransport) { t.healthPath = path }
}

// NewSSE creates a new SSETransport with the given options.
func NewSSE(opts ...SSEOption) *SSETransport {
	t := &SSETransport{
		addr:        ":8080",
		sseEndpoint: "/sse",
		msgEndpoint: "/messages",
		logger:      slog.Default(),
		healthPath:  "/health",
	}
	for _, opt := range opts {
		opt(t)
	}
	t.sessions = newSessionManager(t.logger)
	return t
}

// Start begins serving HTTP and blocks until ctx is canceled or a fatal error occurs.
func (t *SSETransport) Start(ctx context.Context, handler MessageHandler) error {
	mux := http.NewServeMux()

	mux.HandleFunc(t.sseEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		t.handleSSE(ctx, w, r)
	})

	mux.HandleFunc(t.msgEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		t.handleMessages(ctx, w, r, handler)
	})

	if t.healthPath != "" {
		mux.HandleFunc(t.healthPath, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	}

	var h http.Handler = mux
	for i := len(t.httpMws) - 1; i >= 0; i-- {
		h = t.httpMws[i](h)
	}

	t.httpServer = &http.Server{
		Addr:              t.addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		var err error
		if t.tlsCert != "" && t.tlsKey != "" {
			err = t.httpServer.ListenAndServeTLS(t.tlsCert, t.tlsKey)
		} else {
			err = t.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return t.Close()
	case err := <-errCh:
		return err
	}
}

// Send broadcasts a server-initiated message to all connected SSE clients.
func (t *SSETransport) Send(_ context.Context, msg json.RawMessage) error {
	t.sessions.Broadcast(msg)
	return nil
}

// Close performs a graceful shutdown of the HTTP server.
//
//nolint:contextcheck // Close uses an internal timeout context by design.
func (t *SSETransport) Close() error {
	if t.httpServer == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := t.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}
	return nil
}

func (t *SSETransport) authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	if t.auth == nil {
		return ctx, true
	}
	authCtx, err := t.auth.Authenticate(ctx, r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return ctx, false
	}
	return authCtx, true
}

func (t *SSETransport) handleSSE(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, ok := t.authenticate(ctx, w, r)
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	s := t.sessions.Create()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Send initial endpoint event with the POST URL.
	endpointData := fmt.Sprintf(`{"endpoint":"%s?sessionId=%s"}`, t.msgEndpoint, s.id)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointData)
	flusher.Flush()

	done, cleanup := s.addWriter(w, flusher)
	defer cleanup()

	select {
	case <-done:
	case <-r.Context().Done():
	case <-ctx.Done():
	}
}

func (t *SSETransport) handleMessages(ctx context.Context, w http.ResponseWriter, r *http.Request, handler MessageHandler) {
	authCtx, ok := t.authenticate(ctx, w, r)
	if !ok {
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId query parameter", http.StatusBadRequest)
		return
	}

	s := t.sessions.Get(sessionID)
	if s == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp := handler(authCtx, json.RawMessage(body))

	// Write response as SSE event on the session's SSE writer, not in the POST response.
	if resp != nil {
		s.writeSSE(resp)
	}

	w.WriteHeader(http.StatusAccepted)
}
