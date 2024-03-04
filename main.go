package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type RevProxy struct {
	context context.Context
	target  *url.URL
	proxy   *httputil.ReverseProxy
}

func (rp *RevProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Do anything you want here
	// e.g. blacklisting IP, log time, modify headers, etc

	log.Printf("Proxy receives request.")
	log.Printf("Proxy forwards request to origin.")

	req.Host = rp.target.Host
	// rp.recordRequest(r)
	rp.proxy.ServeHTTP(rw, req)

	log.Printf("Origin server completes request.")
}

func (rp *RevProxy) recordRequest(req *http.Request) {
	header := req.Header

	var reqBody []byte
	if body, err := io.ReadAll(req.Body); err == nil {
		reqBody = body
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		log.Println(reqBody)
	}

	cloneHeader := header.Clone()

	headerJSON, err := json.Marshal(cloneHeader)
	if err != nil {
		log.Println("Marshal header failed", "err", err)
		return
	}

	slog.Info("Record request",
		slog.Int64("timestamp", time.Now().Unix()),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("query", req.URL.RawQuery),
		slog.String("header", string(headerJSON)),
		slog.String("Body", string(reqBody)),
	)
}

func modifyRequest(req *http.Request) {
	req.Header.Set("X-Proxy", "Simple-Reverse-Proxy")
	log.Println("--------- modify Request -----------")
}

func modifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		resp.Header.Set("X-Proxy", "Magical")
		log.Println("--------- modify Response -----------")
		return nil
	}
}

func NewRevProxy(ctx context.Context, rawUrl string) (*RevProxy, error) {
	remote, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	s := &RevProxy{
		context: ctx,
		target:  remote,
		proxy:   httputil.NewSingleHostReverseProxy(remote),
	}

	// Modify requests
	originalDirector := s.proxy.Director
	s.proxy.Director = func(r *http.Request) {
		originalDirector(r)
		modifyRequest(r)
	}

	// Modify response
	s.proxy.ModifyResponse = modifyResponse()

	return s, nil
}

func main() {
	// Set the logger for the application
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	revProxy, err := NewRevProxy(context.Background(), "http://localhost:9000")
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: revProxy,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error while shutting down Server. Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
