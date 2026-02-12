package sdk

import (
	"bytes"
	"context"
	"encoding/binary"

	// "encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func Init(parent context.Context) (context.Context, *client.Client, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)

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

func PullImage(ctx context.Context, apiclient *client.Client, imageName string) error {
	image_ref, err := apiclient.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	defer image_ref.Close()
	slog.Info("Pulling image....")
	io.Copy(os.Stdout, image_ref)
	return nil
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

func CreateContainer(ctx context.Context, apiclient *client.Client, containerName string, imageName string, command []string) (string, error) {
	// Create a container from the image
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
	container, err := apiclient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image: imageName,
		Name:  containerName,
		Config: &container.Config{
			Cmd:  command,
			Tty:  false,
			User: "1000:1000",
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	slog.Info("container created",
		"id", container.ID,
		"container_name", containerName,
		"image", imageName,
	)

	return container.ID, nil
}

func StartContainer(ctx context.Context, apiclient *client.Client, containerID string) error {
	// Start the container
	_, err := apiclient.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	slog.Info("container started", "id", containerID)
	return nil
}

func StopContainer(ctx context.Context, apiclient *client.Client, containerID string) error {
	// Stop the container
	timeout := 10
	_, err := apiclient.ContainerStop(ctx, containerID, client.ContainerStopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	slog.Info("container stopped", "id", containerID)
	return nil
}

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

func StatsContainer(ctx context.Context, apiclient *client.Client, containerID string) ([]byte, error) {
	// Get container stats
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

func WaitContainer(ctx context.Context, cli *client.Client, containerID string) (int64, error) {
	statusCh := cli.ContainerWait(ctx, containerID, client.ContainerWaitOptions{})
	select {
	case err := <-statusCh.Error:
		return 0, err
	case status := <-statusCh.Result:
		return status.StatusCode, nil
	}
}
