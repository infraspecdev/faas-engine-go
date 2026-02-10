package api

import (
	"encoding/json"
	"net/http"
)

type DeployResponse struct {
	Message string `json:"message"`
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	resp := DeployResponse{}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Invalid File Size", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")

	if err != nil {
		http.Error(w, "missing 'file' form field", http.StatusBadRequest)
		return
	}

	defer file.Close()

	// , err := io.ReadAll(file)
	// if err != nil {
	// 	http.Error(w, "failed to read file", http.StatusInternalServerError)
	// 	return
	// }

	// fmt.Print(string(data))

	resp.Message = "File received successfully"

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
