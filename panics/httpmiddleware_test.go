package panics

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmateusp/opengym/log"
)

func TestPanicsCatcherMiddleware_NoPanic(t *testing.T) {
	handler := PanicsCatcherMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got '%s'", recorder.Body.String())
	}
}

func TestPanicsCatcherMiddleware_WithPanic(t *testing.T) {
	// Create a buffer to capture logs
	var logBuffer bytes.Buffer
	opts := &slog.HandlerOptions{}
	handler := slog.NewTextHandler(&logBuffer, opts)
	testLogger := slog.New(handler)

	// Create middleware with a handler that panics
	middleware := PanicsCatcherMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	// Create request with logger in context
	ctx := context.Background()
	ctx = log.WithLogger(ctx, testLogger)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)

	// This should not panic
	middleware.ServeHTTP(recorder, req)

	// Should return 500
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	// Check that panic was logged
	logOutput := logBuffer.String()
	if logOutput == "" {
		t.Error("expected panic to be logged, but no log output found")
	}

	if !bytes.Contains(logBuffer.Bytes(), []byte("Panic recovered")) {
		t.Errorf("expected log to contain 'Panic recovered', got: %s", logOutput)
	}
}

func TestPanicsCatcherMiddleware_WithDifferentPanicType(t *testing.T) {
	// Create a buffer to capture logs
	var logBuffer bytes.Buffer
	opts := &slog.HandlerOptions{}
	handler := slog.NewTextHandler(&logBuffer, opts)
	testLogger := slog.New(handler)

	// Create middleware with a handler that panics with an integer
	middleware := PanicsCatcherMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	}))

	// Create request with logger in context
	ctx := context.Background()
	ctx = log.WithLogger(ctx, testLogger)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)

	// This should not panic
	middleware.ServeHTTP(recorder, req)

	// Should return 500
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}
