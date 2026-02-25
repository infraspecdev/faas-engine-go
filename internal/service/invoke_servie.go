package service

import (
	"context"
	"faas-engine-go/internal/sdk"
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
			sdk.StopContainer(cleanupCtx, cli, containerId)
			sdk.DeleteContainer(cleanupCtx, cli, containerId)
		}()
	}()

	if err := sdk.StartContainer(ctx, cli, containerId); err != nil {
		return nil, err
	}

	time.Sleep(50 * time.Millisecond)

	inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}

	port, err := network.ParsePort("8080/tcp")
	if err != nil {
		return nil, err
	}

	bindings := inspect.Container.NetworkSettings.Ports[port]
	hostPort := bindings[0].HostPort

	return sdk.InvokeContainer(ctx, hostPort, payload)
}
