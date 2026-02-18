package api

import (
	"encoding/json"
	"faas-engine-go/internal/sdk"
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

	file, header, err := r.FormFile("file")
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

	fmt.Println("Received file:", header.Filename)

	ctx, cli, cancel, err := sdk.Init(r.Context())
	if err != nil {
		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
		return
	}
	defer cancel()

	err = sdk.CheckImageName(ctx, cli, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = sdk.BuildImage(ctx, cli, name, file)

	if err != nil {
		http.Error(w, "failed to build image", http.StatusInternalServerError)
		return
	}

	resp := DeployResponse{
		Message: fmt.Sprintf("File received successfully and built as image '%s'", name),
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
