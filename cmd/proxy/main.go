package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/proxy"
)

func main() {

	runtimeURL := os.Getenv("RUNTIME_URL")
	if runtimeURL == "" {
		runtimeURL = "http://localhost:8080"
	}

	targetURL, err := url.Parse(runtimeURL)
	if err != nil {
		log.Fatalf("invalid runtime URL: %v", err)
	}

	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "80"
	}

	// Read timeout from env, fallback to constant if not set
	timeoutSecs := int(config.DefaultProxyTimeout.Seconds())
	if timeoutStr := os.Getenv("PROXY_TIMEOUT_SECS"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeoutSecs = t
		}
	}

	slog.Info("configured runtime",
		"runtimeURL", runtimeURL,
		"proxyPort", proxyPort,
		"timeoutSecs", timeoutSecs,
	)

	handler := proxy.ProxyHandler(targetURL, timeoutSecs)

	slog.Info("starting gateway",
		"port", proxyPort,
		"runtime", runtimeURL,
	)

	err = http.ListenAndServe(":"+proxyPort, handler)
	if err != nil {
		slog.Error("gateway error", "error", err)
		log.Fatal(err)
	}
}
