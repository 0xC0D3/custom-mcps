package transport

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
	closed  bool // protected by session.mu
}

type session struct {
	id      string
	writers []*sseWriter
	mu      sync.Mutex
	created time.Time
}

// writeSSE sends an SSE event to all active writers in the session.
func (s *session) writeSSE(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := s.writers[:0]
	for _, sw := range s.writers {
		if sw.closed {
			continue
		}
		if _, err := fmt.Fprintf(sw.w, "data: %s\n\n", data); err != nil {
			sw.closed = true
			continue
		}
		sw.flusher.Flush()
		active = append(active, sw)
	}
	s.writers = active
}

// addWriter registers a new SSE writer for the session.
// It returns a done channel (closed when the session is deleted) and a
// cleanup function that must be called when the HTTP handler returns to
// prevent writes to the closed connection.
func (s *session) addWriter(w http.ResponseWriter, flusher http.Flusher) (<-chan struct{}, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sw := &sseWriter{w: w, flusher: flusher, done: make(chan struct{})}
	s.writers = append(s.writers, sw)
	return sw.done, func() {
		s.mu.Lock()
		sw.closed = true
		s.mu.Unlock()
	}
}

type sessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*session
	logger   *slog.Logger
}

func newSessionManager(logger *slog.Logger) *sessionManager {
	return &sessionManager{
		sessions: make(map[string]*session),
		logger:   logger,
	}
}

// Create creates a new session with a random hex ID.
func (sm *sessionManager) Create() *session {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("transport: failed to generate session ID: " + err.Error())
	}
	id := hex.EncodeToString(b)

	s := &session{
		id:      id,
		created: time.Now(),
	}

	sm.mu.Lock()
	sm.sessions[id] = s
	sm.mu.Unlock()

	sm.logger.Debug("session created", "session_id", id)
	return s
}

// Get returns the session with the given ID, or nil if not found.
func (sm *sessionManager) Get(id string) *session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

// Delete removes the session and closes all its SSE writers.
func (sm *sessionManager) Delete(id string) {
	sm.mu.Lock()
	s, ok := sm.sessions[id]
	if ok {
		delete(sm.sessions, id)
	}
	sm.mu.Unlock()

	if ok {
		s.mu.Lock()
		for _, sw := range s.writers {
			sw.closed = true
			select {
			case <-sw.done:
			default:
				close(sw.done)
			}
		}
		s.mu.Unlock()
		sm.logger.Debug("session deleted", "session_id", id)
	}
}

// Broadcast writes data to all active SSE writers across all sessions.
func (sm *sessionManager) Broadcast(data []byte) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, s := range sm.sessions {
		s.writeSSE(data)
	}
}
