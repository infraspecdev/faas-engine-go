package sdk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/moby/moby/client"
)

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

func BuildImage(ctx context.Context, apiclient *client.Client, imageName string, tarfile io.Reader) error {

	err := CheckImageName(ctx, apiclient, imageName)
	if err != nil {
		return fmt.Errorf("failed to check image name: %w", err)
	}
	slog.Info("Image name check result", "result", "image name is available")

	image, err := apiclient.ImageBuild(ctx, tarfile, client.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
	})

	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	io.Copy(os.Stdout, image.Body)

	slog.Info("Image built successfully", "image_name", imageName)
	return nil
}

func CheckImageName(ctx context.Context, apiclient *client.Client, imageName string) error {
	images, err := apiclient.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images.Items {

		for _, tag := range img.RepoTags {

			// Example: docker.io/library/alpine:latest

			// Remove version tag
			nameWithoutTag := strings.Split(tag, ":")[0]

			// Extract last segment
			parts := strings.Split(nameWithoutTag, "/")
			existingName := parts[len(parts)-1]

			if existingName == imageName {
				return fmt.Errorf("image name '%s' already exists", imageName)
			}
		}
	}
	return nil
}
