package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	jsonMask "github.com/bolom009/go-json-mask"

	"github.com/zjsvv/goreverseproxy/config"
	"github.com/zjsvv/goreverseproxy/middleware"
)

var (
	// -4 means DEBUG; 0 means INFO; 4 means WARN; 8 means ERROR
	logLevelPtr = flag.Int("log_level", 0, "the severity of a log event")
	proxyPort   = flag.String("proxy_port", ":8080", "the exposed port of this proxy server")

	getConfig = config.GetConfig
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
	req.Host = rp.target.Host
	rp.proxy.ServeHTTP(w, req)
}

func shouldBlockRequest(req *http.Request) bool {
	config := getConfig()

	// check if any forbidden header exists
	for header := range req.Header {
		if config.IsHeaderBlocked(header) {
			slog.Debug("[RevProxy][shouldBlockRequest]", slog.String("blockedHeader", header))
			return true
		}
	}

	// check if any forbidden query parameters exists
	for param := range req.URL.Query() {
		if config.IsQueryParamBlocked(param) {
			slog.Debug("[RevProxy][shouldBlockRequest]", slog.String("blockedQueryParam", param))
			return true
		}
	}

	return false
}

func isJSONBody(bodyBytes []byte) bool {
	// try to unmarshal the body into a generic structure
	var js json.RawMessage
	err := json.Unmarshal(bodyBytes, &js)

	return err == nil
}

func maskSensitiveInfo(data string) (string, error) {
	mask := jsonMask.NewJSONMask(getConfig().MaskedNeededKeys...)
	mask.RegisterMaskStringFunc(jsonMask.MaskFilledString("*"))

	maskedData, err := mask.Mask(data)
	if err != nil {
		return "", err
	}
	slog.Debug("[RevProxy][maskSensitiveInfo]",
		slog.String("originalData", data),
		slog.String("maskedData", maskedData),
	)

	return maskedData, nil
}

func modifyResponse(r *http.Response) error {
	originalContentLength := r.ContentLength

	// read the response body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read response body", slog.String("error", err.Error()))
		return err
	}

	// only mask json response body
	if isJSONBody(bodyBytes) {
		// mask sensitive data
		maskedData, err := maskSensitiveInfo(string(bodyBytes))
		if err != nil {
			slog.Error("Failed to mask sensitive information", slog.String("error", err.Error()))
			return err
		}

		// reassign the modified body
		buf := bytes.NewBufferString(maskedData)
		r.Body = io.NopCloser(buf)

		// update Content-Length header
		modifiedContentLength := buf.Len()
		r.Header.Set("Content-Length", strconv.Itoa(modifiedContentLength))

		slog.Debug("[RevProxy][modifyResponse]",
			slog.Int64("originalContentLength", originalContentLength),
			slog.Int("modifiedContentLength", modifiedContentLength),
		)
	} else {
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	return nil
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
	s.proxy.ModifyResponse = modifyResponse

	return s, nil
}

func getLogLevel(logLevelFlag int) slog.Leveler {
	switch {
	case logLevelFlag >= int(slog.LevelError):
		return slog.LevelError
	case logLevelFlag >= int(slog.LevelWarn):
		return slog.LevelWarn
	case logLevelFlag >= int(slog.LevelInfo):
		return slog.LevelInfo
	case logLevelFlag >= int(slog.LevelDebug):
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func main() {
	flag.Parse()

	// set a text logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: getLogLevel(*logLevelPtr)}))
	slog.SetDefault(logger)

	// create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// init config
	config.InitConfig()

	cfg := getConfig()

	revProxy, err := NewRevProxy(context.Background(), cfg.TargetUrl+":"+cfg.TargetPort)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    *proxyPort,
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
