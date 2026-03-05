package sdk

import "github.com/moby/moby/client"

type DockerClient struct {
	cli *client.Client
}

func NewDockerClient(cli *client.Client) *DockerClient {
	return &DockerClient{
		cli: cli,
	}
}

var _ ContainerClient = (*DockerClient)(nil)
var _ ImageClient = (*DockerClient)(nil)
