package api

import (
	"encoding/json"
	"net/http"

	"faas-engine-go/internal/db"
)

func ListInvocationsHandler(w http.ResponseWriter, r *http.Request) {
	invocations := db.ListInvocations()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invocations)
}
