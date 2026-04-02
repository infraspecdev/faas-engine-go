package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type FunctionVersionGetter interface {
	GetVersions(name string) ([]service.VersionInfo, error)
}

// FunctionVersionsHandler is an HTTP handler function that retrieves the versions of a function.
// The function expects the "functionName" parameter in the URL path.
// It returns a JSON response with the versions of the function, or an error response if the function is not found or an internal server error occurs.
func FunctionVersionsHandler(svc FunctionVersionGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		name := strings.TrimSpace(vars["functionName"])
		if name == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		versions, err := svc.GetVersions(name)
		if err != nil {
			if errors.Is(err, service.ErrFunctionNotFound) {
				http.Error(w, "function not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(versions); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
