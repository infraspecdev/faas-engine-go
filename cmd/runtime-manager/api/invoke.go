package api

import (
	"io"
	"net/http"
)

func InvokeHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read payload", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
