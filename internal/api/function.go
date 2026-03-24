package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type FunctionVersionGetter interface {
	GetVersions(name string) ([]service.VersionInfo, error)
}

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

			if strings.Contains(strings.ToLower(err.Error()), "function not found") {
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
