package service

import (
	"context"
	"faas-engine-go/internal/sdk"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type FunctionInvoker struct{}

func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte) (any, error) {

	ctx, cli, cancel, err := sdk.Init(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	target := "localhost:5000/functions/" + functionName

	if err := sdk.PullImage(ctx, cli, target); err != nil {
		return nil, err
	}

	containerId, err := sdk.CreateContainer(ctx, cli, functionName, target, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		go func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := sdk.StopContainer(cleanupCtx, cli, containerId); err != nil {
				fmt.Println("Error stopping container:", err)
			}
		}()
	}()

	if err := sdk.StartContainer(ctx, cli, containerId); err != nil {
		return nil, err
	}

	port, err := network.ParsePort("8080/tcp")

	var hostPort string

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{})

		if err == nil && inspect.Container.NetworkSettings != nil {
			bindings := inspect.Container.NetworkSettings.Ports[port]
			if len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}
	}

	healthDeadline := time.Now().Add(10 * time.Second)
	healthy := false

	for time.Now().Before(healthDeadline) {
		inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{})
		// fmt.Println("container id:", containerId)
		// fmt.Println("Container Health Status:", inspect.Container.State.Health.Status)
		// fmt.Println("container state:", inspect.Container.State.Health.Log)
		if err == nil &&
			inspect.Container.State != nil &&
			inspect.Container.State.Health != nil &&
			inspect.Container.State.Health.Status == "healthy" {
			healthy = true
			break
		}
	}

	if !healthy {
		return nil, fmt.Errorf("container did not become healthy in time")
	}

	return sdk.InvokeContainer(ctx, hostPort, payload)
}
