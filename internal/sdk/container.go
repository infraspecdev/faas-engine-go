package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"faas-engine-go/internal/config"
	"net/http"
	"net/netip"
	"time"

	// "encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// Init initializes a Docker API client with a bounded context.
// It creates a background context (no timeout) for the client lifetime,
// and only applies a timeout to the initialization phase itself.
// The caller must defer the returned cancel function to avoid leaks.
func Init(parent context.Context) (context.Context, *client.Client, context.CancelFunc, error) {
	apiclient, err := client.New(
		client.FromEnv,
		client.WithAPIVersionFromEnv(),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Return a background context (no timeout) for the client's lifetime
	// This allows long-running operations like log streaming to work indefinitely
	ctx, cancel := context.WithCancel(context.Background())
	return ctx, apiclient, cancel, nil
}

// CreateContainer creates a Docker container from the given image.
// If a container with the same name already exists, it returns the existing container ID.
// The container is configured with port 8080 exposed and automatically removed on stop.
// Returns the container ID on success.
func (d *DockerClient) CreateContainer(
	ctx context.Context,
	containerName string,
	imageName string,
	command []string,
) (string, error) {

	// out, err := d.cli.ContainerList(ctx, client.ContainerListOptions{
	// 	All: true,
	// })
	// if err != nil {
	// 	return "", fmt.Errorf("failed to list containers: %w", err)
	// }

	if imageName == "" {
		return "", fmt.Errorf("image name cannot be empty")
	}
	containerName = fmt.Sprintf("%s-%d", containerName, time.Now().UnixNano())

	containerPort, err := network.ParsePort(config.ContainerPort)
	if err != nil {
		return "", fmt.Errorf("failed to parse port: %w", err)
	}

	options := client.ContainerCreateOptions{
		Config: &container.Config{
			Image: imageName,
			Cmd:   command,
			Tty:   false,
			User:  config.ContainerUser,
			ExposedPorts: network.PortSet{
				containerPort: struct{}{},
			},
			Labels: map[string]string{
				"faas-engine": "true",
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{
				containerPort: []network.PortBinding{
					{
						HostIP:   netip.IPv4Unspecified(),
						HostPort: "",
					},
				},
			},
		},
		Name: containerName,
	}

	cont, err := d.cli.ContainerCreate(ctx, options)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return cont.ID, nil
}

// StartContainer starts a previously created Docker container.
// It returns an error if the container fails to start
func (d *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	_, err := d.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// StopContainer gracefully stops a running Docker container.
// It sends a stop signal and waits up to 10 seconds before force termination.
// Returns an error if the container cannot be stopped.
func (d *DockerClient) StopContainer(ctx context.Context, containerID string) error {

	timeout := int(config.ContainerStopTimeout.Seconds())

	_, err := d.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// DeleteContainer removes a Docker container.
// Uses force removal to ensure containers are deleted even if they didn't stop cleanly.
// This is important during graceful shutdown when containers may not respond to signals.
func (d *DockerClient) DeleteContainer(ctx context.Context, containerID string) error {

	_, err := d.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force: true,
	})

	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil
}

// StatsContainer retrieves resource usage statistics for a container.
// It returns the raw JSON stats payload as a byte slice.
// The stats stream is non-continuous (Stream=false).
func (d *DockerClient) StatsContainer(ctx context.Context, containerID string) ([]byte, error) {

	stats, err := d.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream: false,
	})
	if err != nil {
		slog.Error("failed to get container stats",
			"container_id", containerID,
			"error", err,
		)
		return nil, err
	}

	defer func() {
		if err := stats.Body.Close(); err != nil {
			slog.Error(
				"failed to close stats body",
				"error", err,
			)
		}
	}()
	var buf bytes.Buffer

	_, err = io.Copy(&buf, stats.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WaitContainer blocks until the specified container stops.
// It returns the container's exit status code or an error if waiting fails.
func (d *DockerClient) WaitContainer(ctx context.Context, containerID string) (int64, error) {

	statusCh := d.cli.ContainerWait(ctx, containerID, client.ContainerWaitOptions{})

	select {
	case err := <-statusCh.Error:
		return 0, err

	case status := <-statusCh.Result:
		return status.StatusCode, nil
	}
}

func (d *DockerClient) InspectContainer(
	ctx context.Context,
	containerID string,
) (client.ContainerInspectResult, error) {

	return d.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
}

// InvokeContainer sends an HTTP POST request to the running container instance.
// It forwards the provided JSON payload and expects a JSON response.
// Returns a decoded JSON map or an error if the request fails or the container
// returns a non-200 status.
func (d *DockerClient) InvokeContainer(ctx context.Context, hostPort string, body []byte) (map[string]any, error) {

	url := fmt.Sprintf("http://localhost:%s/", hostPort)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{
		Timeout: config.InvokeHTTPTimeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call container: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body",
				"error", err,
			)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("container returned status %d: %s",
			resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode container JSON: %w", err)
	}

	return result, nil
}

func (d *DockerClient) LogContainer(ctx context.Context, containerID string) (string, error) {

	reader, err := d.cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer

	_, err = io.Copy(&buf, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	clean := cleanDockerLogs(buf.Bytes())
	return clean, nil
}

func cleanDockerLogs(raw []byte) string {
	var result []byte

	for i := 0; i < len(raw); {
		if len(raw[i:]) < 8 {
			break
		}

		length := int(raw[i+4])<<24 |
			int(raw[i+5])<<16 |
			int(raw[i+6])<<8 |
			int(raw[i+7])

		i += 8

		if i+length > len(raw) {
			break
		}

		result = append(result, raw[i:i+length]...)
		i += length
	}

	return string(result)
}

func (d *DockerClient) StreamContainerLogs(
	ctx context.Context,
	containerID string,
) (io.ReadCloser, error) {

	// Use the Docker client's background context instead of the HTTP request context
	// to avoid premature cancellation when the request ends
	reader, err := d.cli.ContainerLogs(d.ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "10",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs: %w", err)
	}

	return reader, nil
}

// ListContainers returns all faas-engine labeled containers (running, stopped, and exited)
func (d *DockerClient) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	result, err := d.cli.ContainerList(ctx, client.ContainerListOptions{
		All: true, // Include stopped and exited containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerInfos []ContainerInfo
	filtered := 0
	for _, c := range result.Items {
		// Filter to only include containers with the faas-engine label
		if c.Labels["faas-engine"] == "true" {
			containerInfos = append(containerInfos, ContainerInfo{
				ID:    c.ID,
				State: string(c.State),
			})
		} else {
			filtered++
		}
	}

	if len(result.Items) > 0 && len(containerInfos) > 0 {
		slog.Debug("containers listed", "total", len(result.Items), "labeled", len(containerInfos), "filtered_out", filtered)
	}

	return containerInfos, nil
}
