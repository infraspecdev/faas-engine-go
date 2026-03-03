package sdk

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"faas-engine-go/internal/config"
	"net/http"
	"net/netip"

	// "encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// Init initializes a Docker API client with a bounded context.
// It creates a child context with a timeout and returns
// the context, Docker client, and cancel function.
// The caller must defer the returned cancel function to avoid leaks.
func Init(parent context.Context) (context.Context, *client.Client, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, config.InitTimeout)

	apiclient, err := client.New(
		client.FromEnv,
		client.WithAPIVersionFromEnv(),
	)
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return ctx, apiclient, cancel, nil
}

// func ListContainers(ctx context.Context, cli *client.Client) {
// 	result, err := cli.ContainerList(ctx, client.ContainerListOptions{
// 		All: true,
// 	})
// 	fmt.Println("Listing names of the container")
// 	fmt.Println(result.Items[0].Names)
// 	if err != nil {
// 		fmt.Println("error listing containers:", err)
// 	}

// 	fmt.Println("Converting result to JSON...")
// 	fmt.Println("Listing containers...")

// 	b, err := json.MarshalIndent(result, "", "  ")
// 	if err != nil {
// 		fmt.Println("Error converting result to JSON:", err)
// 	}

// 	fmt.Println(string(b))
// }

// CreateContainer creates a Docker container from the given image.
// If a container with the same name already exists, it returns the existing container ID.
// The container is configured with port 8080 exposed and automatically removed on stop.
// Returns the container ID on success.
func CreateContainer(ctx context.Context, apiclient *client.Client, containerName string, imageName string, command []string) (string, error) {
	out, err := apiclient.ContainerList(ctx, client.ContainerListOptions{
		All: true,
	})

	if imageName == "" {
		return "", fmt.Errorf("image name cannot be empty")
	}

	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	containers := out.Items

	for _, container := range containers {
		if slices.Contains(container.Names, "/"+containerName) {
			slog.Info("container already exists",
				"name", containerName,
				"id", container.ID,
			)

			return container.ID, nil
		}
	}

	containerPort, err := network.ParsePort(config.ContainerPort)

	if err != nil {
		return "", fmt.Errorf("failed to parse port: %w", err)
	}

	options := client.ContainerCreateOptions{
		Config: &container.Config{
			Image: imageName,
			Tty:   false,
			User:  config.ContainerUser,
			ExposedPorts: network.PortSet{
				containerPort: struct{}{},
			},
		},

		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{
				containerPort: []network.PortBinding{
					{
						HostIP:   netip.IPv4Unspecified(),
						HostPort: "", //0.0.0.0:random:8080
					},
				},
			}, AutoRemove: true,
		},

		Name: containerName,
	}

	cont, err := apiclient.ContainerCreate(ctx, options)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return cont.ID, nil

}

// StartContainer starts a previously created Docker container.
// It returns an error if the container fails to start
func StartContainer(ctx context.Context, apiclient *client.Client, containerID string) error {
	_, err := apiclient.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

// StopContainer gracefully stops a running Docker container.
// It sends a stop signal and waits up to 10 seconds before force termination.
// Returns an error if the container cannot be stopped.
func StopContainer(ctx context.Context, apiclient *client.Client, containerID string) error {
	timeout := int(config.ContainerStopTimeout.Seconds())
	_, err := apiclient.ContainerStop(ctx, containerID, client.ContainerStopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// DeleteContainer removes a stopped Docker container.
// It does not force removal and returns an error if deletion fails.
func DeleteContainer(ctx context.Context, cli *client.Client, containerID string) error {
	_, err := cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force: false,
	})

	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	slog.Info("container deleted", "id", containerID)
	return nil
}

// StatsContainer retrieves resource usage statistics for a container.
// It returns the raw JSON stats payload as a byte slice.
// The stats stream is non-continuous (Stream=false).
func StatsContainer(ctx context.Context, apiclient *client.Client, containerID string) ([]byte, error) {
	stats, err := apiclient.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream: false,
	})

	if err != nil {
		slog.Error("failed to get container stats",
			"container_id", containerID,
			"error", err,
		)
		return nil, err
	}

	defer stats.Body.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, stats.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// LogContainer retrieves combined stdout and stderr logs from a container.
// It decodes Docker's multiplexed log stream format and returns plain text logs.
func LogContainer(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	out, err := cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer out.Close()

	var result bytes.Buffer
	header := make([]byte, 8)

	for {
		// Read 8-byte header
		_, err := io.ReadFull(out, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Docker frame format:
		// Byte 0: stream type (1=stdout, 2=stderr)
		// Bytes 1-3: unused
		// Bytes 4-7: payload length (big endian uint32)

		length := binary.BigEndian.Uint32(header[4:])
		if length == 0 {
			continue
		}

		// Read payload
		payload := make([]byte, length)
		_, err = io.ReadFull(out, payload)
		if err != nil {
			return "", err
		}

		result.Write(payload)
	}

	return result.String(), nil
}

// WaitContainer blocks until the specified container stops.
// It returns the container's exit status code or an error if waiting fails.
func WaitContainer(ctx context.Context, cli *client.Client, containerID string) (int64, error) {
	statusCh := cli.ContainerWait(ctx, containerID, client.ContainerWaitOptions{})
	select {
	case err := <-statusCh.Error:
		return 0, err
	case status := <-statusCh.Result:
		return status.StatusCode, nil
	}
}

// InvokeContainer sends an HTTP POST request to the running container instance.
// It forwards the provided JSON payload and expects a JSON response.
// Returns a decoded JSON map or an error if the request fails or the container
// returns a non-200 status.
func InvokeContainer(ctx context.Context, hostPort string, body []byte) (map[string]any, error) {

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
	defer resp.Body.Close()

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
