package server

import (
	"log/slog"
	"time"

	"github.com/0xC0D3/custom-mcps/framework/auth"
	"github.com/0xC0D3/custom-mcps/framework/middleware"
	"github.com/0xC0D3/custom-mcps/framework/transport"
)

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithName sets the server name used in the initialize response.
func WithName(name string) ServerOption {
	return func(s *Server) { s.name = name }
}

// WithVersion sets the server version used in the initialize response.
func WithVersion(version string) ServerOption {
	return func(s *Server) { s.version = version }
}

// WithTransport sets the transport used by the server to communicate with clients.
func WithTransport(t transport.Transport) ServerOption {
	return func(s *Server) { s.transport = t }
}

// WithLogger sets the logger used by the server. Defaults to slog.Default().
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// WithAuthenticator sets the authenticator used by the server.
// Defaults to auth.Noop().
func WithAuthenticator(a auth.Authenticator) ServerOption {
	return func(s *Server) { s.auth = a }
}

// WithMiddleware appends MCP-level middleware to the server's middleware chain.
func WithMiddleware(mws ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.middlewares = append(s.middlewares, mws...) }
}

// WithHTTPMiddleware stores HTTP middleware for HTTP-based transports.
// These are passed through to the transport if it supports them.
func WithHTTPMiddleware(mws ...middleware.HTTPMiddleware) ServerOption {
	return func(s *Server) { s.httpMiddlewares = append(s.httpMiddlewares, mws...) }
}

// WithTLS sets the TLS certificate and key files for HTTP transports.
func WithTLS(certFile, keyFile string) ServerOption {
	return func(s *Server) {
		s.tlsCertFile = certFile
		s.tlsKeyFile = keyFile
	}
}

// WithAddress sets the listen address for HTTP transports.
func WithAddress(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// WithGracefulShutdown sets the timeout for graceful shutdown.
func WithGracefulShutdown(timeout time.Duration) ServerOption {
	return func(s *Server) { s.shutdownTimeout = timeout }
}

// WithInstructions sets the instructions field returned in the initialize response.
func WithInstructions(instructions string) ServerOption {
	return func(s *Server) { s.instructions = instructions }
}
