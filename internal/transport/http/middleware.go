package http

import (
	"log/slog"
	"net/http"
	"time"
)

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r.Context())

		log := s.log.With(
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)
		log.Info("request started")

		t1 := time.Now()

		next.ServeHTTP(w, r)

		log.Info("request completed",
			slog.String("duration", time.Since(t1).String()),
		)
	})
}
