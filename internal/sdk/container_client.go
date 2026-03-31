package sdk

import (
	"context"

	"github.com/moby/moby/client"
)

// ContainerInfo represents minimal information about a container
type ContainerInfo struct {
	ID    string
	State string // running, stopped, exited, etc.
}

type ContainerClient interface {
	CreateContainer(ctx context.Context, name, image string, cmd []string) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
	StatsContainer(ctx context.Context, containerID string) ([]byte, error)
	WaitContainer(ctx context.Context, containerID string) (int64, error)

	InspectContainer(ctx context.Context, containerID string) (client.ContainerInspectResult, error)
	ListContainers(ctx context.Context) ([]ContainerInfo, error)

	LogContainer(ctx context.Context, containerID string) (string, error)
}
