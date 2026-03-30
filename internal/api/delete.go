package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type DeleteResponse struct {
	Message string   `json:"message"`
	Failed  []string `json:"failed,omitempty"`
}

type FunctionDeleter interface {
	DeleteFunction(name string) ([]string, error)
}

// DeleteFunctionHandler handles HTTP requests to delete a function by name.
// It expects the "functionName" path parameter and returns JSON with success or failure details.
func DeleteFunctionHandler(svc FunctionDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		name := strings.TrimSpace(vars["functionName"])
		if name == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		failed, err := svc.DeleteFunction(name)
		if err != nil {

			if errors.Is(err, service.ErrFunctionNotFound) {
				http.Error(w, "function not found", http.StatusNotFound)
				return
			}

			if len(failed) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				if err = json.NewEncoder(w).Encode(DeleteResponse{
					Message: "failed to delete some versions",
					Failed:  failed,
				}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(DeleteResponse{
			Message: "function deleted successfully",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
