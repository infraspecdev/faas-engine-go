package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"faas-engine-go/internal/sdk"
	"log/slog"

	"github.com/gorilla/mux"
)

// lambda invoke greet --data '{"name": "World"}'

// /functions/{functionName}/invoke

// function name : alpine and command : {"echo" : "Hello, World!"}
type invokeReq struct {
	Cmd string `json:"cmd"`
}

func InvokeHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	functionName := strings.TrimSpace(vars["functionName"])
	if functionName == "" {
		http.Error(w, "functionName is required", http.StatusBadRequest)
		return
	}

	slog.Info("invoking function", "name", functionName)

	var req invokeReq

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	cmd := strings.TrimSpace(req.Cmd)
	if cmd == "" {
		http.Error(w, "cmd is required", http.StatusBadRequest)
		return
	}

	ctx, cli, cancel, err := sdk.Init(r.Context())

	if err != nil {
		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
		return
	}
	defer cancel()

	err = sdk.PullImage(ctx, cli, functionName)
	if err != nil {
		http.Error(w, "failed to pull image", http.StatusInternalServerError)
		return
	}

	args := []string{"sh", "-c", req.Cmd}

	containerId, err := sdk.CreateContainer(ctx, cli, functionName, functionName, args)

	if err != nil {
		http.Error(w, "failed to create container", http.StatusInternalServerError)
		return
	}
	slog.Debug("container created", "id", containerId)

	//ensure that the container is stopped and deleted after execution
	defer func(id string) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sdk.StopContainer(cleanupCtx, cli, id); err != nil {
			slog.Error("cleanup failed", "operation", "stop", "container", id, "error", err)
		}

		if err := sdk.DeleteContainer(cleanupCtx, cli, id); err != nil {
			slog.Error("cleanup failed", "operation", "delete", "container", id, "error", err)
		}
	}(containerId)

	err = sdk.StartContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to start container: %v", err), http.StatusInternalServerError)
		return
	}

	statuscode, err := sdk.WaitContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, fmt.Sprintf("container execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	if statuscode != 0 {
		http.Error(w, fmt.Sprintf("container execution failed with status code: %d", statuscode), http.StatusUnprocessableEntity)
		return
	}

	logs, err := sdk.LogContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve container logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(logs))
}
