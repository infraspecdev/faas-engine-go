package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	// Load .env
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env file, using default configuration")
	}

	runtimeUrl := os.Getenv("RUNTIME_URL")
	if runtimeUrl == "" {
		runtimeUrl = "http://localhost"
	}
	runtimePort := os.Getenv("RUNTIME_PORT")
	if runtimePort == "" {
		runtimePort = "8080"
	}

	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "80"
	}

	targetURL, err := url.Parse(runtimeUrl + ":" + runtimePort)
	if err != nil {
		slog.Error("invalid runtime url", "error", err)
		os.Exit(1)
	}

	// Setup router
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(proxyHandler(targetURL))

	// Create server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Run server in background
	go func() {
		slog.Info("starting proxy server", "port", port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("proxy exited gracefully")
	}
}

// proxyHandler returns an HTTP handler that proxies requests to the target URL.
func proxyHandler(targetURL *url.URL) http.HandlerFunc {

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return func(w http.ResponseWriter, r *http.Request) {

		host := r.Host

		proxy.Director = func(req *http.Request) {

			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

			if strings.HasPrefix(host, "localhost") {
				return
			}

			fn := strings.Split(host, ".")[0]
			req.URL.Path = "/functions/" + fn + "/invoke"

			// If GET request → convert to POST with JSON body
			if r.Method == http.MethodGet {

				params := map[string]interface{}{}

				for k, v := range r.URL.Query() {
					if len(v) == 0 {
						continue
					}

					val := v[0]

					// try converting to number
					if num, err := strconv.ParseFloat(val, 64); err == nil {
						params[k] = num
					} else {
						params[k] = val
					}
				}

				body, err := json.Marshal(params)
				if err == nil {
					req.Method = http.MethodPost
					req.Body = io.NopCloser(bytes.NewBuffer(body))
					req.Header.Set("Content-Type", "application/json")
					req.ContentLength = int64(len(body))
				}
			}
		}

		proxy.ServeHTTP(w, r)
	}
}
