package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"

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

	slog.Info("configured runtime",
		"runtimeURL", runtimeURL,
	)

	handler := proxy.ProxyHandler(targetURL)

	slog.Info("starting gateway",
		"port", 80,
		"runtime", runtimeURL,
	)

	err = http.ListenAndServe(":80", handler)
	if err != nil {
		slog.Error("gateway error", "error", err)
		log.Fatal(err)
	}
}
