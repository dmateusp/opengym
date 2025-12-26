package log

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func AddLoggerToContextMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithLogger(r.Context(), logger.With(slog.String("request_id", uuid.NewString())))))
		})
	}
}

const maxCapturedResponseBytes = 64 * 1024

type responseRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	buf          bytes.Buffer
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	if rw.buf.Len() < maxCapturedResponseBytes && n > 0 {
		remaining := maxCapturedResponseBytes - rw.buf.Len()
		if remaining > 0 {
			toCopy := n
			if toCopy > remaining {
				toCopy = remaining
			}
			rw.buf.Write(b[:toCopy])
		}
	}
	return n, err
}

// Preserve optional interfaces
func (rw *responseRecorder) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

func (rw *responseRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func LogRequestsAndResponsesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := FromCtx(r.Context())
		var payload string

		if r.Body != nil {
			// Read and then restore the request body so downstream handlers can read it again
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.WarnContext(r.Context(), "Failed to read request body",
					slog.String("error", err.Error()),
				)
			} else {
				payload = string(bodyBytes)
			}
			// Always restore the body for the next handler
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		logger.InfoContext(r.Context(), "Request received",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("host", r.Host),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("payload", payload),
		)
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rr, r)

		respPayload := rr.buf.String()
		logger.InfoContext(r.Context(), "Response sent",
			slog.Int("status", rr.status),
			slog.Int64("bytes", rr.bytesWritten),
			slog.String("content_type", rr.Header().Get("Content-Type")),
			slog.Duration("duration", time.Since(start)),
			slog.String("payload", respPayload),
		)
	})
}
