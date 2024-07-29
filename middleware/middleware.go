package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"
)

// struct for holding response details
type responseData struct {
	status int
	size   int
	body   *bytes.Buffer
}

// custom http.ResponseWriter implementation
type loggingResponseWriter struct {
	http.ResponseWriter // compose original http.ResponseWriter
	responseData        *responseData
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b) // write response using original http.ResponseWriter
	lrw.responseData.size += size            // capture size
	lrw.responseData.body.Write(b)
	return size, err
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.ResponseWriter.WriteHeader(statusCode) // write status code using original http.ResponseWriter
	lrw.responseData.status = statusCode       // capture status code
}

func (lrw *loggingResponseWriter) Header() http.Header {
	return lrw.ResponseWriter.Header()
}

// Logger is a middleware handler that does request logging
type Logger struct {
	Handler http.Handler
}

// ServeHTTP handles the request by passing it to the real
// handler and logging the request details
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	responseData := &responseData{
		status: 0,
		size:   0,
		body:   bytes.NewBuffer(nil),
	}
	lrw := loggingResponseWriter{
		ResponseWriter: w, // compose original http.ResponseWriter
		responseData:   responseData,
	}

	recordRequest(r)

	slog.Debug("[middleware][ServeHTTP]")
	l.Handler.ServeHTTP(&lrw, r)

	recordResponse(lrw, time.Since(start))
}

// NewLogger constructs a new Logger middleware handler
func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

func recordRequest(req *http.Request) {
	header := req.Header

	var reqBody []byte
	if body, err := io.ReadAll(req.Body); err != nil {
		reqBody = body
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		log.Println(reqBody)
	}

	cloneHeader := header.Clone()

	headerJSON, err := json.Marshal(cloneHeader)
	if err != nil {
		log.Printf("json.Marshal header failed. err: %+s\n", err)
	}

	slog.Debug("[middleware][recordRequest]")
	slog.Info("Record request",
		slog.Int64("timestamp", time.Now().Unix()),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("query", req.URL.RawQuery),
		slog.String("header", string(headerJSON)),
		slog.String("Body", string(reqBody)),
	)
}

func recordResponse(lrw loggingResponseWriter, duration time.Duration) {
	headerJSON, err := json.Marshal(lrw.Header())
	if err != nil {
		log.Printf("json.Marshal header failed. err: %+s\n", err)
	}

	slog.Debug("[middleware] [recordResponse]")
	slog.Info("Request completed",
		slog.Int("status", lrw.responseData.status),
		slog.Int("size", lrw.responseData.size),
		slog.Int64("duration(ms)", duration.Milliseconds()),
		slog.String("header", string(headerJSON)),
		slog.String("body", string(lrw.responseData.body.String())),
	)
}
