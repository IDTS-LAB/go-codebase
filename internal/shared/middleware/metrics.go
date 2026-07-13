package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/monitoring/domain"
)

const (
	MetricRequestsTotal   = "http_requests_total"
	MetricRequestDuration = "http_request_duration_seconds"
	MetricRequestsActive  = "http_requests_active"
	MetricRequestSize     = "http_request_size_bytes"
	MetricResponseSize    = "http_response_size_bytes"
)

func Metrics(recorder domain.MetricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder.IncrementCounter(r.Context(), MetricRequestsActive, "method", r.Method, "path", r.URL.Path, "status", "0")
			defer recorder.IncrementCounter(r.Context(), MetricRequestsActive, "method", r.Method, "path", r.URL.Path, "status", "0")

			if r.ContentLength > 0 {
				recorder.ObserveHistogram(r.Context(), MetricRequestSize, float64(r.ContentLength), "method", r.Method, "path", r.URL.Path, "status", "0")
			}

			wrapped := &metricsWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r.WithContext(r.Context()))

			status := strconv.Itoa(wrapped.statusCode)
			duration := time.Since(start).Seconds()

			recorder.IncrementCounter(r.Context(), MetricRequestsTotal, "method", r.Method, "path", r.URL.Path, "status", status)
			recorder.ObserveHistogram(r.Context(), MetricRequestDuration, duration, "method", r.Method, "path", r.URL.Path, "status", status)
			if wrapped.bodySize > 0 {
				recorder.ObserveHistogram(r.Context(), MetricResponseSize, float64(wrapped.bodySize), "method", r.Method, "path", r.URL.Path, "status", status)
			}
		})
	}
}

type metricsWriter struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func (w *metricsWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bodySize += n
	return n, err
}
