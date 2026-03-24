package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Invoker interface {
	Invoke(ctx context.Context, functionName string, payload []byte) (any, error)
}

func InvokeHandler(invoker Invoker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		result, err := invoker.Invoke(r.Context(), functionName, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Error("failed to encode invoke response", "error", err)
		}
	}
}
