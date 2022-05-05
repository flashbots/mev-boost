package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

func makePostRequest(ctx context.Context, client http.Client, url string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return resp, fmt.Errorf("HTTP error status code: %d", resp.StatusCode)
	}

	return resp, nil
}

type httpResponseContainer struct {
	url  string
	err  error
	resp *http.Response
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// LoggingMiddleware logs the incoming HTTP request & its duration.
func LoggingMiddleware(next http.Handler, log *logrus.Entry) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)

					method := ""
					url := ""
					if r != nil {
						method = r.Method
						url = r.URL.EscapedPath()
					}

					log.Info(fmt.Sprintf("http request panic: %s %s", method, url),
						"err", err,
						"trace", string(debug.Stack()),
					)
				}
			}()
			start := time.Now()
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			log.WithFields(logrus.Fields{
				"status":     wrapped.status,
				"method":     r.Method,
				"path":       r.URL.EscapedPath(),
				"durationMs": time.Since(start).Milliseconds(),
			}).Info(fmt.Sprintf("http: %s %s %d", r.Method, r.URL.EscapedPath(), wrapped.status))
		},
	)
}