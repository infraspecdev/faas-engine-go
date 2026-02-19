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
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Missing function name", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	ctx, cli, cancel, err := sdk.Init(ctx)
	if err != nil {
		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
		return
	}
	defer cancel()

	deployer := service.Deployer{CLI: cli}

	if err := deployer.Deploy(ctx, name, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(DeployResponse{
		Message: fmt.Sprintf("Deployed '%s' successfully", name),
	})
}
