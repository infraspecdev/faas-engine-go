package api

import (
	"encoding/json"
	"net/http"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

func FunctionVersionsHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	name := vars["functionName"]

	versions, err := service.GetFunctionVersions(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}
