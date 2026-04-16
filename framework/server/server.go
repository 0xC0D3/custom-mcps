// Package server provides the top-level MCP server builder and runtime.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/0xC0D3/custom-mcps/framework/auth"
	mw "github.com/0xC0D3/custom-mcps/framework/middleware"
	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/0xC0D3/custom-mcps/framework/registry"
	"github.com/0xC0D3/custom-mcps/framework/transport"
)

// Server is the top-level MCP server. It composes a transport, registry,
// authenticator, middleware chain, and handles JSON-RPC dispatch.
type Server struct {
	name            string
	version         string
	transport       transport.Transport
	registry        *registry.Registry
	auth            auth.Authenticator
	logger          *slog.Logger
	middlewares     []mw.Middleware
	httpMiddlewares []mw.HTTPMiddleware
	instructions    string
	initialized     bool
	tlsCertFile     string
	tlsKeyFile      string
	address         string
	shutdownTimeout time.Duration
}

// New creates a new Server with sensible defaults and applies the given options.
func New(opts ...Option) *Server {
	s := &Server{
		name:     "mcp-server",
		version:  "1.0.0",
		registry: registry.New(),
		auth:     auth.Noop(),
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.transport == nil {
		s.transport = transport.NewStdio()
	}
	return s
}

// Registry returns the server's registry for direct registration of tools,
// resources, and prompts.
func (s *Server) Registry() *registry.Registry {
	return s.registry
}

// Tool is a package-level generic helper that registers a typed tool handler
// on the server's registry. This is the primary registration API for tools.
func Tool[T any](s *Server, name, description string, handler func(ctx context.Context, input T) (*protocol.CallToolResult, error)) {
	registry.RegisterTool[T](s.registry, name, description, handler)
}

// Run starts the server and blocks until the context is canceled or a fatal
// error occurs. It builds the middleware chain and starts the transport.
func (s *Server) Run(ctx context.Context) error {
	// Build the handler chain. Start with dispatch as the innermost handler.
	var base mw.MessageHandler = func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return s.dispatch(ctx, raw)
	}

	// Apply user middleware (innermost, closest to dispatch).
	if len(s.middlewares) > 0 {
		chain := mw.Chain(s.middlewares...)
		base = chain(base)
	}

	// Apply built-in middleware (outermost: recovery first, then logging).
	base = mw.Chain(
		mw.Recovery(s.logger),
		mw.Logging(s.logger),
	)(base)

	// Bridge middleware.MessageHandler to transport.MessageHandler.
	// They have the same underlying signature, so a direct type conversion works.
	handler := transport.MessageHandler(base)

	if err := s.transport.Start(ctx, handler); err != nil {
		return fmt.Errorf("running transport: %w", err)
	}
	return nil
}

// dispatch is the core JSON-RPC router. It parses incoming messages and routes
// them to the appropriate handler based on the method name.
func (s *Server) dispatch(ctx context.Context, raw json.RawMessage) json.RawMessage {
	msg, err := protocol.ParseMessage(raw)
	if err != nil {
		var rpcErr *protocol.RPCError
		if errors.As(err, &rpcErr) {
			return mustMarshal(protocol.NewErrorResponse(nil, rpcErr))
		}
		return mustMarshal(protocol.NewErrorResponse(nil, protocol.NewInternalError(err)))
	}

	switch m := msg.(type) {
	case *protocol.Notification:
		return s.handleNotification(m)
	case *protocol.Request:
		return s.handleRequest(ctx, m)
	default:
		// Responses or unknown types are ignored.
		return nil
	}
}

func (s *Server) handleNotification(n *protocol.Notification) json.RawMessage {
	switch n.Method {
	case "notifications/initialized":
		s.initialized = true
	case "notifications/cancelled": //nolint:misspell // MCP protocol method name uses British spelling
		s.logger.Info("request canceled by client")
	default:
		s.logger.Info("unknown notification", slog.String("method", n.Method))
	}
	return nil
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Request) json.RawMessage {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "ping":
		return s.respond(req.ID, struct{}{})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "prompts/list":
		return s.handlePromptsList(req)
	case "prompts/get":
		return s.handlePromptsGet(ctx, req)
	default:
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewMethodNotFound(req.Method)))
	}
}

func (s *Server) handleInitialize(req *protocol.Request) json.RawMessage {
	var params protocol.InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInvalidParams(err.Error())))
		}
	}

	caps := protocol.ServerCapabilities{}
	if tools := s.registry.ListTools(); len(tools) > 0 {
		caps.Tools = &protocol.ToolCapability{}
	}
	if resources := s.registry.ListResources(); len(resources) > 0 {
		caps.Resources = &protocol.ResourceCapability{}
	}
	if prompts := s.registry.ListPrompts(); len(prompts) > 0 {
		caps.Prompts = &protocol.PromptCapability{}
	}

	result := protocol.InitializeResult{
		ProtocolVersion: protocol.ProtocolVersion,
		Capabilities:    caps,
		ServerInfo: protocol.Implementation{
			Name:    s.name,
			Version: s.version,
		},
		Instructions: s.instructions,
	}

	return s.respond(req.ID, result)
}

func (s *Server) handleToolsList(req *protocol.Request) json.RawMessage {
	tools := s.registry.ListTools()
	result := protocol.ToolListResult{Tools: tools}
	return s.respond(req.ID, result)
}

func (s *Server) handleToolsCall(ctx context.Context, req *protocol.Request) json.RawMessage {
	var params protocol.CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInvalidParams(err.Error())))
	}

	tool := s.registry.GetTool(params.Name)
	if tool == nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID,
			protocol.NewInvalidParams("unknown tool: "+params.Name)))
	}

	result, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		var rpcErr *protocol.RPCError
		if errors.As(err, &rpcErr) {
			return mustMarshal(protocol.NewErrorResponse(req.ID, rpcErr))
		}
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInternalError(err)))
	}

	return s.respond(req.ID, result)
}

func (s *Server) handleResourcesList(req *protocol.Request) json.RawMessage {
	resources := s.registry.ListResources()
	result := protocol.ResourceListResult{Resources: resources}
	return s.respond(req.ID, result)
}

func (s *Server) handleResourcesRead(ctx context.Context, req *protocol.Request) json.RawMessage {
	var params protocol.ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInvalidParams(err.Error())))
	}

	resource := s.registry.GetResource(params.URI)
	if resource == nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID,
			protocol.NewInvalidParams("unknown resource: "+params.URI)))
	}

	result, err := resource.Handler(ctx, params.URI)
	if err != nil {
		var rpcErr *protocol.RPCError
		if errors.As(err, &rpcErr) {
			return mustMarshal(protocol.NewErrorResponse(req.ID, rpcErr))
		}
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInternalError(err)))
	}

	return s.respond(req.ID, result)
}

func (s *Server) handlePromptsList(req *protocol.Request) json.RawMessage {
	prompts := s.registry.ListPrompts()
	result := protocol.PromptListResult{Prompts: prompts}
	return s.respond(req.ID, result)
}

func (s *Server) handlePromptsGet(ctx context.Context, req *protocol.Request) json.RawMessage {
	var params protocol.GetPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInvalidParams(err.Error())))
	}

	prompt := s.registry.GetPrompt(params.Name)
	if prompt == nil {
		return mustMarshal(protocol.NewErrorResponse(req.ID,
			protocol.NewInvalidParams("unknown prompt: "+params.Name)))
	}

	result, err := prompt.Handler(ctx, params.Arguments)
	if err != nil {
		var rpcErr *protocol.RPCError
		if errors.As(err, &rpcErr) {
			return mustMarshal(protocol.NewErrorResponse(req.ID, rpcErr))
		}
		return mustMarshal(protocol.NewErrorResponse(req.ID, protocol.NewInternalError(err)))
	}

	return s.respond(req.ID, result)
}

// respond builds a successful JSON-RPC response and marshals it.
func (s *Server) respond(id json.RawMessage, result any) json.RawMessage {
	resp, err := protocol.NewResponse(id, result)
	if err != nil {
		s.logger.Error("failed to marshal response", slog.Any("error", err))
		return mustMarshal(protocol.NewErrorResponse(id, protocol.NewInternalError(err)))
	}
	return mustMarshal(resp)
}

// mustMarshal marshals v to JSON, panicking on failure (should not happen for
// well-formed protocol types).
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("server: failed to marshal JSON: " + err.Error())
	}
	return data
}
