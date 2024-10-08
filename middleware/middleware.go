package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

var (
	jsonMarshal = json.Marshal
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

	l.Handler.ServeHTTP(&lrw, r)

	recordResponse(lrw, time.Since(start))
}

// NewLogger constructs a new Logger middleware handler
func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

func recordRequest(req *http.Request) {
	// create a new reader that simultaneously reads data from a source reader and write the same data to a writer
	copy := new(bytes.Buffer)
	req.Body = io.NopCloser(io.TeeReader(req.Body, copy))

	// everything read from req.Body will be copied to copy
	data, err := io.ReadAll(req.Body)
	if err != nil {
		slog.Error("Error reading from request body", slog.String("err", err.Error()))
		return
	}

	// assign the copied buffer to request body to let next handler handle the request body
	req.Body = io.NopCloser(copy)

	// get headers
	headers := composeRequestHeaders(req)

	headersJSON, err := jsonMarshal(headers)
	if err != nil {
		slog.Error("jsonMarshal header failed", slog.String("err", err.Error()))
		return
	}

	slog.Info("Record request",
		slog.Int64("timestamp", time.Now().Unix()),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("query", req.URL.RawQuery),
		slog.String("headers", string(headersJSON)),
		slog.String("body", string(data)),
	)
}

func recordResponse(lrw loggingResponseWriter, duration time.Duration) {
	headersJSON, err := jsonMarshal(lrw.Header())
	if err != nil {
		slog.Error("jsonMarshal header failed", slog.String("err", err.Error()))
	}

	slog.Info("Request completed",
		slog.Int("status", lrw.responseData.status),
		slog.Int("size", lrw.responseData.size),
		slog.Int64("duration(ms)", duration.Milliseconds()),
		slog.String("headers", string(headersJSON)),
		slog.String("body", string(lrw.responseData.body.String())),
	)
}

func composeRequestHeaders(req *http.Request) map[string][]string {
	if req == nil {
		return make(map[string][]string)
	}

	headers := make(map[string][]string)

	cloneHeader := req.Header.Clone()

	for key, val := range cloneHeader {
		headers[key] = val
	}

	headers["Content-Length"] = []string{strconv.Itoa(int(req.ContentLength))}
	headers["Host"] = []string{req.Host}

	return headers
}
