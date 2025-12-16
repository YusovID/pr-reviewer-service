package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Метрика общего количества запросов (Counter)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	// Метрика времени выполнения запросов (Histogram)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
)

// responseWriterWrapper нужен для перехвата статус-кода ответа
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{w, http.StatusOK}
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware — middleware для сбора метрик
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := newResponseWriterWrapper(w)
		next.ServeHTTP(wrapper, r)

		duration := time.Since(start).Seconds()

		// Записываем метрики
		statusCode := strconv.Itoa(wrapper.statusCode)

		// Используем route pattern (если возможно) или r.URL.Path
		// Примечание: в чистом chi сложно получить pattern в middleware без дополнительных усилий,
		// поэтому для простоты используем Path. В продакшене лучше группировать ID.
		path := r.URL.Path

		httpRequestsTotal.WithLabelValues(path, r.Method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(path, r.Method).Observe(duration)
	})
}
