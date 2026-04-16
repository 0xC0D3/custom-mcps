package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// StdioTransport implements Transport over stdin/stdout using
// newline-delimited JSON messages.
type StdioTransport struct {
	in     io.Reader
	out    io.Writer
	mu     sync.Mutex
	logger *slog.Logger
}

// StdioOption configures a StdioTransport.
type StdioOption func(*StdioTransport)

// WithStdioInput sets the reader for incoming messages.
func WithStdioInput(r io.Reader) StdioOption {
	return func(t *StdioTransport) { t.in = r }
}

// WithStdioOutput sets the writer for outgoing messages.
func WithStdioOutput(w io.Writer) StdioOption {
	return func(t *StdioTransport) { t.out = w }
}

// WithStdioLogger sets the logger for the transport.
func WithStdioLogger(l *slog.Logger) StdioOption {
	return func(t *StdioTransport) { t.logger = l }
}

// NewStdio creates a new StdioTransport with the given options.
// Defaults to os.Stdin and os.Stdout.
func NewStdio(opts ...StdioOption) *StdioTransport {
	t := &StdioTransport{
		in:     os.Stdin,
		out:    os.Stdout,
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Start reads newline-delimited JSON from the input, dispatches each message
// to handler, and writes any non-nil response as a single JSON line to the output.
// It blocks until ctx is canceled or the input stream ends.
func (t *StdioTransport) Start(ctx context.Context, handler MessageHandler) error {
	scanner := bufio.NewScanner(t.in)
	// MCP messages can be large; allow up to 1 MB per line.
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if !scanner.Scan() {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		raw := make(json.RawMessage, len(line))
		copy(raw, line)

		resp := handler(ctx, raw)
		if resp != nil {
			if err := t.writeLine(resp); err != nil {
				t.logger.Error("failed to write response", "error", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-ctx.Done():
			return nil
		default:
			t.logger.Error("scanner error", "error", err)
			return fmt.Errorf("reading input: %w", err)
		}
	}

	return nil
}

// Send writes a server-initiated message to the output as a single JSON line.
func (t *StdioTransport) Send(_ context.Context, msg json.RawMessage) error {
	return t.writeLine(msg)
}

// Close is a no-op for stdio transport; the process exiting handles cleanup.
func (t *StdioTransport) Close() error {
	return nil
}

func (t *StdioTransport) writeLine(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.out.Write(data); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if _, err := t.out.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	return nil
}
