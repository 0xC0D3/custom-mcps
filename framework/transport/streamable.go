package transport

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/0xC0D3/custom-mcps/framework/auth"
)

// StreamableHTTPTransport implements Transport using the streamable HTTP
// protocol. It serves a single endpoint that handles POST (requests),
// GET (SSE stream), and DELETE (session teardown).
type StreamableHTTPTransport struct {
	addr       string
	endpoint   string
	httpServer *http.Server
	sessions   *sessionManager
	auth       auth.Authenticator
	httpMws    []HTTPMiddleware
	logger     *slog.Logger
	tlsCert    string
	tlsKey     string
	healthPath string
}

// StreamableOption configures a StreamableHTTPTransport.
type StreamableOption func(*StreamableHTTPTransport)

// WithAddr sets the listen address.
func WithAddr(addr string) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.addr = addr }
}

// WithEndpoint sets the MCP endpoint path.
func WithEndpoint(endpoint string) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.endpoint = endpoint }
}

// WithStreamableAuth sets the authenticator.
func WithStreamableAuth(a auth.Authenticator) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.auth = a }
}

// WithStreamableHTTPMiddleware adds HTTP middlewares.
func WithStreamableHTTPMiddleware(mws ...HTTPMiddleware) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.httpMws = append(t.httpMws, mws...) }
}

// WithStreamableLogger sets the logger.
func WithStreamableLogger(l *slog.Logger) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.logger = l }
}

// WithStreamableTLS enables TLS with the given certificate and key files.
func WithStreamableTLS(certFile, keyFile string) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.tlsCert = certFile; t.tlsKey = keyFile }
}

// WithHealthPath sets the health endpoint path. Empty string disables it.
func WithHealthPath(path string) StreamableOption {
	return func(t *StreamableHTTPTransport) { t.healthPath = path }
}

// NewStreamableHTTP creates a new StreamableHTTPTransport with the given options.
func NewStreamableHTTP(opts ...StreamableOption) *StreamableHTTPTransport {
	t := &StreamableHTTPTransport{
		addr:       ":8080",
		endpoint:   "/mcp",
		logger:     slog.Default(),
		healthPath: "/health",
	}
	for _, opt := range opts {
		opt(t)
	}
	t.sessions = newSessionManager(t.logger)
	return t
}

// Start begins serving HTTP and blocks until ctx is cancelled or a fatal error occurs.
func (t *StreamableHTTPTransport) Start(ctx context.Context, handler MessageHandler) error {
	mux := http.NewServeMux()

	mux.HandleFunc(t.endpoint, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			t.handlePost(ctx, w, r, handler)
		case http.MethodGet:
			t.handleGet(ctx, w, r)
		case http.MethodDelete:
			t.handleDelete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
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
		Addr:    t.addr,
		Handler: h,
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
func (t *StreamableHTTPTransport) Send(_ context.Context, msg json.RawMessage) error {
	t.sessions.Broadcast(msg)
	return nil
}

// Close performs a graceful shutdown of the HTTP server.
func (t *StreamableHTTPTransport) Close() error {
	if t.httpServer == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return t.httpServer.Shutdown(shutdownCtx)
}

// Addr returns the listener address after the server has started.
// This is useful for tests that use ":0" to pick a random port.
func (t *StreamableHTTPTransport) Addr() net.Addr {
	// This method is available only when using the test helper; the
	// standard Start() path uses httpServer.ListenAndServe which
	// doesn't expose the listener. Tests should use httptest.NewServer
	// or set a known port.
	return nil
}

func (t *StreamableHTTPTransport) authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
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

func (t *StreamableHTTPTransport) handlePost(ctx context.Context, w http.ResponseWriter, r *http.Request, handler MessageHandler) {
	authCtx, ok := t.authenticate(ctx, w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get or create session.
	sessionID := r.Header.Get("Mcp-Session-Id")
	var s *session
	if sessionID != "" {
		s = t.sessions.Get(sessionID)
	}
	if s == nil {
		s = t.sessions.Create()
	}

	resp := handler(authCtx, json.RawMessage(body))

	w.Header().Set("Mcp-Session-Id", s.id)
	if resp != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}

func (t *StreamableHTTPTransport) handleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, ok := t.authenticate(ctx, w, r)
	if !ok {
		return
	}

	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		http.Error(w, "Missing Mcp-Session-Id header", http.StatusBadRequest)
		return
	}

	s := t.sessions.Get(sessionID)
	if s == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	done, cleanup := s.addWriter(w, flusher)
	defer cleanup()

	select {
	case <-done:
	case <-r.Context().Done():
	case <-ctx.Done():
	}
}

func (t *StreamableHTTPTransport) handleDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		http.Error(w, "Missing Mcp-Session-Id header", http.StatusBadRequest)
		return
	}

	s := t.sessions.Get(sessionID)
	if s == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	t.sessions.Delete(sessionID)
	w.WriteHeader(http.StatusNoContent)
}
