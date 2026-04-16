package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Logging tests
// ---------------------------------------------------------------------------

func TestLogging_LogsMethodAndDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := Logging(logger)(func(_ context.Context, _ json.RawMessage) json.RawMessage {
		resp, _ := json.Marshal(protocol.NewErrorResponse(json.RawMessage(`1`), protocol.NewInternalError(fmt.Errorf("boom"))))
		return resp
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"test/hello"}`
	handler(context.Background(), json.RawMessage(req))

	logged := buf.String()
	assert.Contains(t, logged, "test/hello", "log should contain the method name")
	assert.Contains(t, logged, "duration", "log should contain the duration")
	assert.Contains(t, logged, "has_error=true", "log should indicate an error response")
}

func TestLogging_NoError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := Logging(logger)(func(_ context.Context, _ json.RawMessage) json.RawMessage {
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "ok"})
		return resp
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"test/ok"}`
	handler(context.Background(), json.RawMessage(req))

	logged := buf.String()
	assert.Contains(t, logged, "test/ok")
	assert.Contains(t, logged, "has_error=false")
}

// ---------------------------------------------------------------------------
// Recovery tests
// ---------------------------------------------------------------------------

func TestRecovery_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := Recovery(logger)(func(_ context.Context, _ json.RawMessage) json.RawMessage {
		panic("something broke")
	})

	req := `{"jsonrpc":"2.0","id":42,"method":"test/panic"}`
	resp := handler(context.Background(), json.RawMessage(req))

	require.NotNil(t, resp, "recovery should return a response")

	var rpcResp protocol.Response
	err := json.Unmarshal(resp, &rpcResp)
	require.NoError(t, err)

	assert.Equal(t, "2.0", rpcResp.JSONRPC)
	assert.Equal(t, json.RawMessage(`42`), rpcResp.ID)
	require.NotNil(t, rpcResp.Error)
	assert.Equal(t, protocol.CodeInternalError, rpcResp.Error.Code)
	assert.Contains(t, rpcResp.Error.Message, "something broke")

	assert.Contains(t, buf.String(), "panic recovered")
}

func TestRecovery_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := Recovery(logger)(func(_ context.Context, _ json.RawMessage) json.RawMessage {
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "ok"})
		return resp
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"test/safe"}`
	resp := handler(context.Background(), json.RawMessage(req))
	require.NotNil(t, resp)
	assert.Empty(t, buf.String(), "no panic should produce no error log")
}

// ---------------------------------------------------------------------------
// Chain tests
// ---------------------------------------------------------------------------

func TestChain_ExecutionOrder(t *testing.T) {
	var order []string

	makeMW := func(name string) Middleware {
		return func(next MessageHandler) MessageHandler {
			return func(ctx context.Context, raw json.RawMessage) json.RawMessage {
				order = append(order, name+"-before")
				resp := next(ctx, raw)
				order = append(order, name+"-after")
				return resp
			}
		}
	}

	chained := Chain(makeMW("A"), makeMW("B"), makeMW("C"))
	handler := chained(func(_ context.Context, _ json.RawMessage) json.RawMessage {
		order = append(order, "handler")
		return nil
	})

	handler(context.Background(), json.RawMessage(`{}`))

	expected := []string{
		"A-before", "B-before", "C-before",
		"handler",
		"C-after", "B-after", "A-after",
	}
	assert.Equal(t, expected, order)
}

func TestChainHTTP_ExecutionOrder(t *testing.T) {
	var order []string

	makeMW := func(name string) HTTPMiddleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-before")
				next.ServeHTTP(w, r)
				order = append(order, name+"-after")
			})
		}
	}

	chained := ChainHTTP(makeMW("X"), makeMW("Y"))
	handler := chained(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	expected := []string{"X-before", "Y-before", "handler", "Y-after", "X-after"}
	assert.Equal(t, expected, order)
}

// ---------------------------------------------------------------------------
// Rate limit tests
// ---------------------------------------------------------------------------

func TestRateLimit_AllowsUpToBurst(t *testing.T) {
	mw := RateLimit(1, 3)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := range 3 {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
	}

	// Next request should be rate limited.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "rate limit exceeded")
}

func TestRateLimit_DifferentIPsAreIndependent(t *testing.T) {
	mw := RateLimit(1, 1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP uses its token.
	req1 := httptest.NewRequest(http.MethodPost, "/", nil)
	req1.RemoteAddr = "10.0.0.1:1111"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second IP should still have its own token.
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.RemoteAddr = "10.0.0.2:2222"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

// ---------------------------------------------------------------------------
// CORS tests
// ---------------------------------------------------------------------------

func TestCORS_RegularRequest(t *testing.T) {
	mw := CORS("https://example.com")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Mcp-Session-Id")
	assert.Equal(t, "Mcp-Session-Id", rec.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORS_OptionsPreflightReturns204(t *testing.T) {
	mw := CORS("*")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://any.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_UnmatchedOriginNoHeaders(t *testing.T) {
	mw := CORS("https://allowed.com")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------------------------------------------------------------------
// Metrics tests
// ---------------------------------------------------------------------------

func TestMetrics_TracksRequestsAndCodes(t *testing.T) {
	mw := Metrics("/metrics")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/fail") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Send 3 OK requests.
	for range 3 {
		req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Send 1 failing request.
	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Fetch metrics.
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	assert.Equal(t, http.StatusOK, metricsRec.Code)
	assert.Equal(t, "application/json", metricsRec.Header().Get("Content-Type"))

	var data metricsData
	err := json.NewDecoder(metricsRec.Body).Decode(&data)
	require.NoError(t, err)

	assert.Equal(t, int64(4), data.TotalRequests)
	assert.Equal(t, int64(3), data.StatusCodes["200"])
	assert.Equal(t, int64(1), data.StatusCodes["500"])
	assert.Greater(t, data.AvgDurationMS, float64(0))
}

func TestMetrics_EndpointNotCountedInMetrics(t *testing.T) {
	mw := Metrics("/metrics")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Only fetch metrics (no other requests).
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	var data metricsData
	err := json.NewDecoder(metricsRec.Body).Decode(&data)
	require.NoError(t, err)

	assert.Equal(t, int64(0), data.TotalRequests, "metrics endpoint itself should not be counted")
}
