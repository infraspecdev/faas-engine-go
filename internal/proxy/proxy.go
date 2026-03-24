package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func ProxyHandler(targetURL *url.URL) http.Handler {

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.Director = nil

	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if strings.Contains(err.Error(), "timeout") {
			http.Error(w, "Function timeout", http.StatusGatewayTimeout)
			return
		}
		slog.Error("proxy error", "error", err)
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

			// safer header copy
			copyHeaders(out.Header, req.Header)

			slog.Info("control plane request", "path", req.URL.Path)
			return
		}

		fn := extractFunctionName(req.Host)

		if fn == "" {
			return // handled in outer handler
		}

		out.URL.Path = "/functions/" + fn + "/invoke"
		out.URL.RawQuery = req.URL.RawQuery

		copyHeaders(out.Header, req.Header)

		slog.Info("invoke request", "function", fn)

		if req.Method == http.MethodGet {

			params := map[string]interface{}{}

			for k, v := range req.URL.Query() {
				if len(v) == 0 {
					continue
				}

				val := v[0]

				if num, err := strconv.ParseFloat(val, 64); err == nil {
					params[k] = num
				} else {
					params[k] = val
				}
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

		// Allow control plane
		if strings.HasPrefix(r.URL.Path, "/functions") {
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

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func extractFunctionName(host string) string {

	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	if strings.HasSuffix(host, ".nip.io") {
		parts := strings.Split(host, ".")
		if len(parts) >= 6 {
			return parts[0]
		}
	}

	if strings.HasSuffix(host, ".localhost") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			return parts[0]
		}
	}

	parts := strings.Split(host, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}
