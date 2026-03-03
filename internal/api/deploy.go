package api

import (
	"context"
	"encoding/json"
	"faas-engine-go/internal/types"
	"fmt"
	"io"
	"net/http"
)

type Deployer interface {
	Deploy(ctx context.Context, name string, file io.Reader) error
}

func DeployHandler(deployer Deployer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

		if err := r.ParseMultipartForm(50 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "file too large",
			})
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing 'file' field",
			})
			return
		}
		defer file.Close()

		name := r.FormValue("name")
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing function name",
			})
			return
		}

		ctx := context.Background()

		if err := deployer.Deploy(ctx, name, file); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, types.DeployResponse{
			Message: fmt.Sprintf("Deployed '%s' successfully", name),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
