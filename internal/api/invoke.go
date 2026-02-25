package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Invoker interface {
	Invoke(ctx context.Context, functionName string, payload []byte) (any, error)
}

// lambda invoke greet --data '{"name": "World"}'

// /functions/{functionName}/invoke

// function name : alpine and command : {"echo" : "Hello, World!"}

// func InvokeHandler(w http.ResponseWriter, r *http.Request) {

// 	vars := mux.Vars(r)
// 	functionName := strings.TrimSpace(vars["functionName"])
// 	if functionName == "" {
// 		http.Error(w, "functionName is required", http.StatusBadRequest)
// 		return
// 	}

// 	slog.Info("invoking function", "name", functionName)

// 	body, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, "failed to read request body", http.StatusBadRequest)
// 		return
// 	}
// 	defer r.Body.Close()

// 	ctx, cli, cancel, err := sdk.Init(r.Context())

// 	if err != nil {
// 		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
// 		return
// 	}
// 	defer cancel()

// 	target := "localhost:5000/functions/" + functionName
// 	err = sdk.PullImage(ctx, cli, target)
// 	if err != nil {
// 		http.Error(w, "failed to pull image", http.StatusInternalServerError)
// 		return
// 	}

// 	containerId, err := sdk.CreateContainer(ctx, cli, functionName, target, nil) // cmd should have been here

// 	if err != nil {
// 		http.Error(w, "failed to create container", http.StatusInternalServerError)
// 		return
// 	}
// 	slog.Debug("container created", "id", containerId)

// 	defer func(id string) {
// 		go func() {
// 			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 			defer cancel()

// 			sdk.StopContainer(cleanupCtx, cli, id)
// 			sdk.DeleteContainer(cleanupCtx, cli, id)
// 		}()
// 	}(containerId)

// 	err = sdk.StartContainer(ctx, cli, containerId)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("failed to start container: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	time.Sleep(20 * time.Millisecond)

// 	inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{
// 		Size: false,
// 	})
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("failed to inspect container: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	port, err := network.ParsePort("8080/tcp")
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("failed to parse port: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	bindings := inspect.Container.NetworkSettings.Ports[port]
// 	hostPort := bindings[0].HostPort

// 	fmt.Printf("hostPort: %+v\n", hostPort)

// 	result, err := sdk.InvokeContainer(ctx, hostPort, body)
// 	if err != nil {
// 		fmt.Print(err)
// 		http.Error(w, fmt.Sprintf("failed to invoke container: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// statuscode, err := sdk.WaitContainer(ctx, cli, containerId)
// 	// if err != nil {
// 	// 	http.Error(w, fmt.Sprintf("container execution failed: %v", err), http.StatusInternalServerError)
// 	// 	return
// 	// }
// 	// if statuscode != 0 {
// 	// 	http.Error(w, fmt.Sprintf("container execution failed with status code: %d", statuscode), http.StatusUnprocessableEntity)
// 	// 	return
// 	// }

// 	// logs, err := sdk.LogContainer(ctx, cli, containerId)
// 	// if err != nil {
// 	// 	http.Error(w, fmt.Sprintf("failed to retrieve container logs: %v", err), http.StatusInternalServerError)
// 	// 	return
// 	// }

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(result)
// }

func InvokeHandler(invoker Invoker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		result, err := invoker.Invoke(r.Context(), functionName, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
