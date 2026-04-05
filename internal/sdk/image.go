package sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

// ListImages retrieves a list of all Docker images available locally.
// It returns a slice of ImageSummary structs containing metadata about each image
func (d *DockerClient) ListImages(ctx context.Context) ([]image.Summary, error) {
	result, err := d.cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return result.Items, nil
}

// PullImage pulls a Docker image from a registry.
// It consumes the entire response stream to ensure the pull completes successfully.
// Returns an error if the pull operation fails.
func (d *DockerClient) PullImage(ctx context.Context, imageName string) error {

	imageRef, err := d.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	defer func() {
		if err := imageRef.Close(); err != nil {
			fmt.Printf("failed to close image reference: %v", err)
		}
	}()

	if _, err := io.Copy(io.Discard, imageRef); err != nil {
		return fmt.Errorf("failed to copy image data: %w", err)
	}

	return nil
}

// BuildImage builds a Docker image using the provided tar build context.
// It validates that the image if name does not already exist before building.
func (d *DockerClient) BuildImage(
	ctx context.Context,
	imageName string,
	tarfile io.Reader,
	out io.Writer,
) error {

	err := d.CheckImageName(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to check image name: %w", err)
	}

	image, err := d.cli.ImageBuild(ctx, tarfile, client.ImageBuildOptions{
		Tags:        []string{imageName},
		Dockerfile:  "Dockerfile",
		Remove:      true,
		ForceRemove: true,
		NoCache:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	defer func() {
		if err := image.Body.Close(); err != nil {
			fmt.Printf("failed to close image body: %v", err)
		}
	}()

	if err := streamDockerLogs(image.Body, out); err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

	_, err = d.cli.ImagePrune(ctx, client.ImagePruneOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CheckImageName verifies that the given image name does not already exist as a local untagged image.
// Only checks for exact local image name matches (not registry-prefixed images).
// Returns an error if a conflicting image name is found.
func (d *DockerClient) CheckImageName(ctx context.Context, imageName string) error {
	images, err := d.cli.ImageList(ctx, client.ImageListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images.Items {
		for _, tag := range img.RepoTags {
			// Only check for local image names (no "/" means it's a local image, not from registry).
			// Example: "calc:latest" or "alpine:3.18" (local)
			// Skip: "localhost:5000/functions/calc:v1" (registry image)
			if !strings.Contains(tag, "/") {
				// Remove version tag to get the image name
				nameWithoutTag := strings.Split(tag, ":")[0]

				if nameWithoutTag == imageName {
					return fmt.Errorf("image name '%s' already exists locally", imageName)
				}
			}
		}
	}
	return nil
}

// PushImage pushes a tagged Docker image to its configured registry.
// Returns an error if the push fails.
func (d *DockerClient) PushImage(ctx context.Context, target string) error {
	username := config.RegistryUsername()
	password := config.RegistryPassword()
	registry := config.Registry()

	opts := client.ImagePushOptions{}

	// Only include registry auth for non-localhost registries.
	// localhost:5000 doesn't require authentication.
	isLocalRegistry := strings.HasPrefix(registry, "localhost") ||
		strings.HasPrefix(registry, "127.0.0.1")

	if !isLocalRegistry && username != "" && password != "" {
		auth := map[string]string{
			"username":      username,
			"password":      password,
			"serveraddress": registry,
		}

		authJSON, err := json.Marshal(auth)
		if err != nil {
			return fmt.Errorf("failed to marshal registry auth: %w", err)
		}

		opts.RegistryAuth = base64.StdEncoding.EncodeToString(authJSON)
	}

	imagePush, err := d.cli.ImagePush(ctx, target, opts)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer func() {
		if err := imagePush.Close(); err != nil {
			fmt.Printf("failed to close image push stream: %v", err)
		}
	}()

	if _, err := io.Copy(io.Discard, imagePush); err != nil {
		return fmt.Errorf("failed to read push output: %w", err)
	}

	return nil
}

// TagImage tags an existing Docker image with a new reference.
// Both source and target should be valid image references (e.g., "alpine:latest", "localhost:5000/functions/calc:v1").
// Returns an error if the source image does not exist or if tagging fails.
func (d *DockerClient) TagImage(ctx context.Context, source string, target string) error {
	_, err := d.cli.ImageTag(ctx, client.ImageTagOptions{
		Source: source,
		Target: target,
	})
	if err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w", source, target, err)
	}
	return nil
}

// RemoveImage removes a local Docker image by reference.
// It does not force removal and will fail if the image is in use.
func (d *DockerClient) RemoveImage(ctx context.Context, target string) error {
	_, err := d.cli.ImageRemove(ctx, target, client.ImageRemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	return nil
}
