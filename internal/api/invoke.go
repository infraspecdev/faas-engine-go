package api

import (
	"encoding/json"
	"faas-engine-go/internal/core"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// InvokeHandler handles HTTP requests for invoking a deployed function.
// It expects a "functionName" path parameter and a request body.
// Returns 400 if input is invalid and 500 if invocation fails.
func InvokeHandler(invoker core.Invoker) http.HandlerFunc {
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

		result, err := invoker.Invoke(r.Context(), functionName, body, "http")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
