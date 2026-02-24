package api

import (
	"context"
	"encoding/json"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/service"
	"fmt"
	"net/http"
)

type DeployResponse struct {
	Message string `json:"message"`
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {

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

	ctx, cli, cancel, err := sdk.Init(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to initialize SDK",
		})
		return
	}
	defer cancel()

	deployer := service.Deployer{CLI: cli}

	if err := deployer.Deploy(ctx, name, file); err != nil {
		fmt.Printf("Error in the deploy handler: %s\n", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, DeployResponse{
		Message: fmt.Sprintf("Deployed '%s' successfully", name),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
