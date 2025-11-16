package http

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := getRequestID(r.Context())

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(id))
		require.NoError(t, err)
	})

	server := &Server{}
	handlerToTest := server.requestID(nextHandler)

	t.Run("Generate new request ID if header is missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://testing", nil)
		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		respHeaderID := rr.Header().Get(requestIDHeader)
		respBodyID := rr.Body.String()

		assert.NotEmpty(t, respHeaderID, "response header should have a request ID")
		assert.NotEmpty(t, respBodyID, "response body should have a request ID from context")
		assert.Equal(t, respHeaderID, respBodyID, "header and context ID should match")
	})

	t.Run("Use existing request ID from header", func(t *testing.T) {
		const existingID = "test-request-id-123"

		req := httptest.NewRequest("GET", "http://testing", nil)
		req.Header.Set(requestIDHeader, existingID)

		rr := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rr, req)

		assert.Equal(t, existingID, rr.Header().Get(requestIDHeader))
		assert.Equal(t, existingID, rr.Body.String())
	})
}

func TestLogRequestMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	server := &Server{log: logger}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := server.requestID(server.logRequest(nextHandler))

	req := httptest.NewRequest("GET", "/test-path", nil)
	rr := httptest.NewRecorder()

	handlerToTest.ServeHTTP(rr, req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "request started", "should log start of request")
	assert.Contains(t, logOutput, "request completed", "should log end of request")
	assert.Contains(t, logOutput, "method=GET", "should log request method")
	assert.Contains(t, logOutput, "path=/test-path", "should log request path")
	assert.Contains(t, logOutput, "duration=", "should log request duration")
	assert.Contains(t, logOutput, "request_id=", "should log request ID")
}

func TestGetRequestID(t *testing.T) {
	t.Run("Returns ID if present in context", func(t *testing.T) {
		const expectedID = "my-test-id"
		ctx := context.WithValue(context.Background(), requestIDKey, expectedID)
		id := getRequestID(ctx)
		assert.Equal(t, expectedID, id)
	})

	t.Run("Returns empty string if not in context", func(t *testing.T) {
		id := getRequestID(context.Background())
		assert.Empty(t, id)
	})
}
