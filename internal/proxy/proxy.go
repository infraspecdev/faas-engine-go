package proxy

import (
	"bytes"
	"encoding/json"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// validFunctionNamePattern allows alphanumeric characters and hyphens only
var validFunctionNamePattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// getProxyTimeout reads PROXY_TIMEOUT_SECS from environment, defaults to the constant
func getProxyTimeout() time.Duration {
	defaultTimeout := config.DefaultProxyTimeout
	if timeoutStr := os.Getenv("PROXY_TIMEOUT_SECS"); timeoutStr != "" {
		if secs, err := strconv.Atoi(timeoutStr); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultTimeout
}

func ProxyHandler(targetURL *url.URL, timeoutSecs ...int) http.Handler {
	timeout := getProxyTimeout()
	// Override with explicit parameter if provided
	if len(timeoutSecs) > 0 && timeoutSecs[0] > 0 {
		timeout = time.Duration(timeoutSecs[0]) * time.Second
	}

	// Create ReverseProxy manually without a default Director.
	// We use Rewrite exclusively, so we don't set Director at all.
	// This avoids the "ReverseProxy must have exactly one of Director or Rewrite set" error.
	proxy := &httputil.ReverseProxy{
		Transport: &http.Transport{
			ResponseHeaderTimeout: timeout,
		},
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// Distinguish between different error types
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			slog.Warn("proxy timeout", "url", r.URL.String(), "error", err)
			http.Error(w, "Function timeout", http.StatusGatewayTimeout)
			return
		}

		// Check for connection refused (runtime not available)
		if strings.Contains(err.Error(), "connection refused") {
			slog.Error("runtime connection refused", "url", r.URL.String(), "error", err)
			http.Error(w, "Runtime unavailable", http.StatusServiceUnavailable)
			return
		}

		// Check for context cancelled (client disconnected)
		if strings.Contains(err.Error(), "context canceled") {
			slog.Warn("request cancelled", "url", r.URL.String(), "error", err)
			http.Error(w, "Request cancelled", http.StatusRequestTimeout)
			return
		}

		// Generic error
		slog.Error("proxy error", "url", r.URL.String(), "error", err)
		http.Error(w, "Runtime error", http.StatusInternalServerError)
	}

	proxy.Rewrite = func(pr *httputil.ProxyRequest) {

		req := pr.In
		out := pr.Out

		// Always target runtime
		pr.SetURL(targetURL)

		slog.Info("incoming request",
			"host", req.Host,
			"path", req.URL.Path,
			"method", req.Method,
		)

		if strings.HasPrefix(req.URL.Path, "/functions") {
			out.URL.Path = req.URL.Path
			out.URL.RawQuery = req.URL.RawQuery

			// Direct assignment instead of Add() to avoid duplicating headers
			for k, vv := range req.Header {
				out.Header[k] = vv
			}

			slog.Info("control plane request", "path", req.URL.Path)
			return
		}

		fn := extractFunctionName(req.Host)

		if fn == "" {
			return // handled in outer handler
		}

		out.URL.Path = "/functions/" + fn + "/invoke"
		out.URL.RawQuery = req.URL.RawQuery

		// Direct assignment instead of Add() to avoid duplicating headers
		for k, vv := range req.Header {
			out.Header[k] = vv
		}

		slog.Info("invoke request", "function", fn)

		if req.Method == http.MethodGet {

			// Treat all query params as strings, not auto-converted to numbers
			// This prevents precision loss and silent type conversion of strings like phone numbers
			params := map[string]interface{}{}

			for k, v := range req.URL.Query() {
				if len(v) == 0 {
					continue
				}

				// Keep as string - no implicit type conversion
				params[k] = v[0]
			}

			body, err := json.Marshal(params)
			if err == nil {
				out.Method = http.MethodPost
				out.Body = io.NopCloser(bytes.NewBuffer(body))
				out.ContentLength = int64(len(body))
				out.Header.Set("Content-Type", "application/json")
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Health check endpoint
		if r.URL.Path == "/__health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"healthy","timeout":"%v"}`, timeout)
			return
		}

		// Allow control plane endpoints
		if strings.HasPrefix(r.URL.Path, "/functions") || strings.HasPrefix(r.URL.Path, "/schedules") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Require function for invoke
		fn := extractFunctionName(r.Host)
		if fn == "" {
			slog.Error("invalid invoke request", "host", r.Host)
			http.Error(w, "Function not specified", http.StatusBadRequest)
			return
		}

		proxy.ServeHTTP(w, r)
	})
}

// isValidFunctionName validates that a function name contains only alphanumeric characters and hyphens
func isValidFunctionName(name string) bool {
	if name == "" {
		return false
	}
	return validFunctionNamePattern.MatchString(name)
}

func extractFunctionName(host string) string {

	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	var fn string

	if strings.HasSuffix(host, ".nip.io") {
		parts := strings.Split(host, ".")
		if len(parts) >= 6 {
			fn = parts[0]
		}
	} else if strings.HasSuffix(host, ".localhost") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			fn = parts[0]
		}
	} else {
		parts := strings.Split(host, ".")
		if len(parts) > 1 {
			fn = parts[0]
		}
	}

	// Validate function name: alphanumeric + hyphens only
	// Prevents path traversal attacks like "../../../etc"
	if !isValidFunctionName(fn) {
		slog.Warn("invalid function name detected", "name", fn, "host", host)
		return ""
	}

	return fn
}
