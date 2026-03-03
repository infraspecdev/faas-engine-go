package api

import (
	"context"
	"encoding/json"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/types"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Deployer interface {
	Deploy(ctx context.Context, name string, file io.Reader) error
}

// DeployHandler handles multipart file upload requests for deploying a new function.
// It enforces a maximum upload size and returns 400 Bad Request if the file field is missing.
func DeployHandler(deployer Deployer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)

		if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
			slog.Error("image_lifecycle",
				"stage", "invalid_upload",
				"error", err,
			)
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "file too large",
			})
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			slog.Error("image_lifecycle",
				"stage", "missing_file",
				"error", err,
			)
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing 'file' field",
			})
			return
		}
		defer file.Close()

		name := r.FormValue("name")
		if name == "" {
			slog.Error("image_lifecycle",
				"stage", "missing_name",
			)
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing function name",
			})
			return
		}

		logger := slog.With("function", name)

		logger.Info("image_lifecycle", "stage", "deploying")

		if err := deployer.Deploy(r.Context(), name, file); err != nil {
			logger.Error("image_lifecycle",
				"stage", "failed",
				"error", err,
			)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		logger.Info("image_lifecycle", "stage", "deployed")

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
