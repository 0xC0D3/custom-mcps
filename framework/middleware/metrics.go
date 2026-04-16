package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// metricsData holds the collected metrics for JSON serialization.
type metricsData struct {
	TotalRequests int64            `json:"total_requests"`
	StatusCodes   map[string]int64 `json:"status_codes"`
	AvgDurationMS float64          `json:"avg_duration_ms"`
}

// metricsCollector accumulates request metrics in a thread-safe manner.
type metricsCollector struct {
	totalRequests atomic.Int64
	statusCodes   sync.Map     // map[int]*atomic.Int64
	totalDuration atomic.Int64 // nanoseconds
}

// record records a completed request with the given status code and duration.
func (m *metricsCollector) record(statusCode int, d time.Duration) {
	m.totalRequests.Add(1)
	m.totalDuration.Add(int64(d))

	val, _ := m.statusCodes.LoadOrStore(statusCode, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}

// snapshot returns the current metrics as a serialisable struct.
func (m *metricsCollector) snapshot() metricsData {
	total := m.totalRequests.Load()
	codes := make(map[string]int64)

	m.statusCodes.Range(func(key, value any) bool {
		code := key.(int)
		count := value.(*atomic.Int64).Load()
		codes[strconv.Itoa(code)] = count
		return true
	})

	var avgMS float64
	if total > 0 {
		avgMS = float64(m.totalDuration.Load()) / float64(total) / float64(time.Millisecond)
	}

	return metricsData{
		TotalRequests: total,
		StatusCodes:   codes,
		AvgDurationMS: avgMS,
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before delegating to the underlying writer.
func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Metrics returns HTTP middleware that tracks request counts and durations.
// The collected metrics are exposed as JSON at the given path (e.g. "/metrics").
// The JSON response includes total_requests, status_codes, and avg_duration_ms.
func Metrics(path string) HTTPMiddleware {
	collector := &metricsCollector{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == path {
				snap := collector.snapshot()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(snap)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)
			collector.record(rec.statusCode, time.Since(start))
		})
	}
}
