package sdk

import (
	"context"

	"github.com/moby/moby/client"
)

type DockerClient struct {
	cli *client.Client
	ctx context.Context
}

func NewDockerClient(cli *client.Client, ctx context.Context) *DockerClient {
	return &DockerClient{
		cli: cli,
		ctx: ctx,
	}
}

var _ ContainerClient = (*DockerClient)(nil)
var _ ImageClient = (*DockerClient)(nil)
