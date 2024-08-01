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
	"github.com/zjsvv/goreverseproxy/config"
)

type RevProxy struct {
	context context.Context
	target  *url.URL
	proxy   *httputil.ReverseProxy
}

func (rp *RevProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// block request if it contains specific headers or parameters
	if req.Method == http.MethodGet && shouldBlockRequest(req) {
		slog.Debug("[RevProxy][ServeHTTP] Blocking request due to specific headers or parameters.")
		http.Error(w, "Request blocked by proxy rules", http.StatusForbidden)
		return
	}

	slog.Debug("[RevProxy][ServeHTTP] Proxy is going to forward request to origin.")

	req.Host = rp.target.Host
	rp.proxy.ServeHTTP(w, req)

	slog.Debug("[RevProxy][ServeHTTP] Origin server completes request.")
}

func shouldBlockRequest(req *http.Request) bool {
	config := config.GetConfig()

	// check for forbidden headers
	if config.IsHeaderBlocked("X-Custom-Key") {
		return true
	}

	// check for forbidden query parameters
	if req.URL.Query().Get("blockedParam") != "" {
		return true
	}
	return false
}

func modifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		slog.Debug("[RevProxy][modifyResponse]")
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

	// customize response
	s.proxy.ModifyResponse = modifyResponse()

	return s, nil
}

func main() {
	// set a text logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// init config
	config.InitConfig()

	origin := "http://localhost:9000"

	revProxy, err := NewRevProxy(context.Background(), origin)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: middleware.NewLogger(revProxy),
	}

	// initializing the server in a goroutine so that it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// listen for the interrupt signal.
	<-ctx.Done()

	// restore default behavior on the interrupt signal and notify user of shutdown
	stop()
	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// the context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error while shutting down Server. Server forced to shutdown: ", err)
	}

	slog.Info("Server exiting")
}
