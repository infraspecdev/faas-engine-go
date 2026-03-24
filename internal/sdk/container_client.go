package sdk

import (
	"context"
	"io"

	"github.com/moby/moby/client"
)

type ContainerClient interface {
	CreateContainer(ctx context.Context, name, image string, cmd []string) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
	StatsContainer(ctx context.Context, containerID string) ([]byte, error)
	WaitContainer(ctx context.Context, containerID string) (int64, error)
	InvokeContainer(ctx context.Context, hostPort string, body []byte) (map[string]any, error)

	InspectContainer(ctx context.Context, containerID string) (client.ContainerInspectResult, error)

	LogContainer(ctx context.Context, containerID string) (string, error)
	StreamContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
}
