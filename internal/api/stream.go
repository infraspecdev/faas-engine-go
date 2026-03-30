package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type LogStreamer interface {
	StreamFunctionLogs(ctx context.Context, functionID int, out chan<- string) error
	GetFunctionID(name string) (int, error)
}

// LogStreamHandler handles HTTP requests for streaming function logs.
// It expects the "functionName" path parameter and sends server-sent events with log lines.
func LogStreamHandler(streamer LogStreamer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		functionID, err := streamer.GetFunctionID(functionName)
		if err != nil {
			http.Error(w, "function not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		logChan := make(chan string, 100)

		go func() {
			err := streamer.StreamFunctionLogs(r.Context(), functionID, logChan)
			if err != nil {
				logChan <- fmt.Sprintf("error: %v", err)
				close(logChan)
			}
		}()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {

			case log, ok := <-logChan:
				if !ok {
					return
				}
				fmt.Fprintf(w, "%s\n\n", log)
				flusher.Flush()

			case <-ticker.C:
				flusher.Flush()

			case <-r.Context().Done():
				return
			}
		}
	}
}
