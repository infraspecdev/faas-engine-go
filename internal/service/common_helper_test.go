package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type fakeLogContainerClient struct {
	fakeContainerClient
	running bool
	logs    []string
}

func (f *fakeLogContainerClient) InspectContainer(ctx context.Context, id string) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{
		Container: container.InspectResponse{
			State: &container.State{
				Running: f.running,
			},
		},
	}, nil
}

func (f *fakeLogContainerClient) StreamContainerLogs(ctx context.Context, id string) (io.ReadCloser, error) {
	return createDockerLogStream(f.logs), nil
}

func createDockerLogStream(lines []string) io.ReadCloser {
	var buf bytes.Buffer

	for _, line := range lines {
		payload := []byte(line)
		header := make([]byte, 8)
		binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
		buf.Write(header)
		buf.Write(payload)
	}

	return io.NopCloser(&buf)
}

type fakeImageClient struct {
	buildCalled  bool
	tagCalled    bool
	pushCalled   bool
	removeCalled bool

	buildErr  error
	tagErr    error
	pushErr   error
	removeErr error

	lastTagSource string
	lastTagTarget string
	lastPushImage string

	pullCalled bool
	pullErr    error
}

func (f *fakeImageClient) PullImage(ctx context.Context, name string) error {
	f.pullCalled = true
	return f.pullErr
}

func (f *fakeImageClient) BuildImage(ctx context.Context, name string, r io.Reader, w io.Writer) error {
	f.buildCalled = true
	return f.buildErr
}

func (f *fakeImageClient) TagImage(ctx context.Context, source, target string) error {
	f.tagCalled = true
	f.lastTagSource = source
	f.lastTagTarget = target
	return f.tagErr
}

func (f *fakeImageClient) PushImage(ctx context.Context, name string) error {
	f.pushCalled = true
	f.lastPushImage = name
	return f.pushErr
}

func (f *fakeImageClient) RemoveImage(ctx context.Context, name string) error {
	f.removeCalled = true
	return f.removeErr
}
