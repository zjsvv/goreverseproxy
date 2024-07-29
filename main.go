package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zjsvv/goreverseproxy/middleware"
)

type RevProxy struct {
	context context.Context
	target  *url.URL
	proxy   *httputil.ReverseProxy
}

func (rp *RevProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	slog.Info("[RevProxy][ServeHTTP] Proxy is going to forward request to origin.")

	req.Host = rp.target.Host
	rp.proxy.ServeHTTP(w, req)

	slog.Info("[RevProxy][ServeHTTP] Origin server completes request.")
}

func modifyRequest(req *http.Request) {
	slog.Info("[RevProxy][modifyRequest]")
	req.Header.Set("X-Proxy", "Simple-Reverse-Proxy")
}

func modifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		slog.Info("[RevProxy][modifyResponse]")
		resp.Header.Set("X-Proxy", "Magical")
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

	origin := "http://localhost:9000"

	revProxy, err := NewRevProxy(context.Background(), origin)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: middleware.NewLogger(revProxy),
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
	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error while shutting down Server. Server forced to shutdown: ", err)
	}

	slog.Info("Server exiting")
}
