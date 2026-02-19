package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"faas-engine-go/internal/sdk"
	"log/slog"

	"github.com/gorilla/mux"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// lambda invoke greet --data '{"name": "World"}'

// /functions/{functionName}/invoke

// function name : alpine and command : {"echo" : "Hello, World!"}

func InvokeHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	functionName := strings.TrimSpace(vars["functionName"])
	if functionName == "" {
		http.Error(w, "functionName is required", http.StatusBadRequest)
		return
	}

	slog.Info("invoking function", "name", functionName)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// cmd := strings.TrimSpace(req.Cmd)
	// if cmd == "" {
	// 	http.Error(w, "cmd is required", http.StatusBadRequest)
	// 	return
	// }

	ctx, cli, cancel, err := sdk.Init(r.Context())

	if err != nil {
		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
		return
	}
	defer cancel()

	err = sdk.PullImage(ctx, cli, "localhost:5000/functions/echo:latest")
	if err != nil {
		http.Error(w, "failed to pull image", http.StatusInternalServerError)
		return
	}

	containerId, err := sdk.CreateContainer(ctx, cli, functionName, "localhost:5000/functions/echo:latest", nil) // cmd should have been here

	if err != nil {
		http.Error(w, "failed to create container", http.StatusInternalServerError)
		return
	}
	slog.Debug("container created", "id", containerId)

	// ensure that the container is stopped and deleted after execution
	defer func(id string) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sdk.StopContainer(cleanupCtx, cli, id); err != nil {
			slog.Error("cleanup failed", "operation", "stop", "container", id, "error", err)
		}

		// if err := sdk.DeleteContainer(cleanupCtx, cli, id); err != nil {
		// 	slog.Error("cleanup failed", "operation", "delete", "container", id, "error", err)
		// }
	}(containerId)

	err = sdk.StartContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to start container: %v", err), http.StatusInternalServerError)
		return
	}

	inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{
		Size: false,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to inspect container: %v", err), http.StatusInternalServerError)
		return
	}

	port, err := network.ParsePort("8080/tcp")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse port: %v", err), http.StatusInternalServerError)
		return
	}

	bindings := inspect.Container.NetworkSettings.Ports[port]
	hostPort := bindings[0].HostPort

	fmt.Printf("hostPort: %+v\n", hostPort)

	result, err := sdk.InvokeContainer(ctx, hostPort, body)
	if err != nil {
		fmt.Print(err)
		http.Error(w, fmt.Sprintf("failed to invoke container: %v", err), http.StatusInternalServerError)
		return
	}

	// statuscode, err := sdk.WaitContainer(ctx, cli, containerId)
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("container execution failed: %v", err), http.StatusInternalServerError)
	// 	return
	// }
	// if statuscode != 0 {
	// 	http.Error(w, fmt.Sprintf("container execution failed with status code: %d", statuscode), http.StatusUnprocessableEntity)
	// 	return
	// }

	// logs, err := sdk.LogContainer(ctx, cli, containerId)
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("failed to retrieve container logs: %v", err), http.StatusInternalServerError)
	// 	return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
