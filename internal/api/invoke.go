package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"faas-engine-go/internal/sdk"

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
	functionName := vars["functionName"]

	fmt.Printf("Invoking function: %s\n", functionName)
	ctx, cli, err := sdk.Init()

	if err != nil {
		http.Error(w, "failed to initialize SDK", http.StatusInternalServerError)
		return
	}

	err = sdk.PullImage(ctx, cli, functionName)
	if err != nil {
		http.Error(w, "failed to pull image", http.StatusInternalServerError)
		return
	}

	var req invokeReq

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	args := []string{"sh", "-c", req.Cmd}

	containerId, err := sdk.CreateContainer(ctx, cli, functionName, functionName, args)

	if err != nil {
		http.Error(w, "failed to create container", http.StatusInternalServerError)
		return
	}
	fmt.Println("Container created with ID:", containerId)
	err = sdk.StartContainer(ctx, cli, containerId)

	if err != nil {
		http.Error(w, "failed to start container", http.StatusInternalServerError)
		return
	}

	_, err = sdk.WaitContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, "wait failed", 500)
		return
	}

	logs, err := sdk.GetContainerLogs(ctx, cli, containerId)
	if err != nil {
		http.Error(w, "logs failed", 500)
		return
	}

	err = sdk.StopContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, "failed to stop container", http.StatusInternalServerError)
		return
	}

	err = sdk.DeleteContainer(ctx, cli, containerId)
	if err != nil {
		http.Error(w, "failed to delete container", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(logs))
}
