package sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/client"
)

// PullImage pulls a Docker image from a registry.
// It consumes the entire response stream to ensure the pull completes successfully.
// Returns an error if the pull operation fails.
func PullImage(ctx context.Context, apiclient *client.Client, imageName string) error {
	imageRef, err := apiclient.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	defer imageRef.Close()
	if _, err := io.Copy(io.Discard, imageRef); err != nil {
		return fmt.Errorf("failed to copy image data: %w", err)
	}
	return nil
}

// BuildImage builds a Docker image using the provided tar build context.
// It validates that the image if name does not already exist before building.
func BuildImage(ctx context.Context, apiclient *client.Client, imageName string, tarfile io.Reader, out io.Writer) error {

	err := CheckImageName(ctx, apiclient, imageName)
	if err != nil {
		return fmt.Errorf("failed to check image name: %w", err)
	}

	image, err := apiclient.ImageBuild(ctx, tarfile, client.ImageBuildOptions{
		Tags:        []string{imageName},
		Dockerfile:  "Dockerfile",
		Remove:      true,
		ForceRemove: true,
		NoCache:     true,
	})

	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer image.Body.Close()

	if err := streamDockerLogs(image.Body, out); err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

	_, err = apiclient.ImagePrune(ctx, client.ImagePruneOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CheckImageName verifies that the given image name does not already exist locally.
// Returns an error if a conflicting image name is found.
func CheckImageName(ctx context.Context, apiclient *client.Client, imageName string) error {
	images, err := apiclient.ImageList(ctx, client.ImageListOptions{
		All: true,
	})
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

// TagImage creates a new tag for an existing Docker image.
// The source image must exist locally.
// Returns an error if tagging fails.
func TagImage(ctx context.Context, apiclient *client.Client, source string, target string) error {
	_, err := apiclient.ImageTag(ctx, client.ImageTagOptions{
		Source: source,
		Target: target,
	})

	if err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}

	return nil
}

// PushImage pushes a tagged Docker image to its configured registry.
// Returns an error if the push fails.
func PushImage(ctx context.Context, cli *client.Client, target string) error {

	auth := map[string]string{
		"username":      "",
		"password":      "",
		"serveraddress": config.Registry(),
	}

	authJSON, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("failed to marshal registry auth: %w", err)
	}

	encodedAuth := base64.StdEncoding.EncodeToString(authJSON)

	imagePush, err := cli.ImagePush(ctx, target, client.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	defer imagePush.Close()

	if _, err := io.Copy(io.Discard, imagePush); err != nil {
		return fmt.Errorf("failed to read push output: %w", err)
	}

	return nil
}

// RemoveImage removes a local Docker image by reference.
// It does not force removal and will fail if the image is in use.
func RemoveImage(ctx context.Context, apiclient *client.Client, target string) error {
	_, err := apiclient.ImageRemove(ctx, target, client.ImageRemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	return nil
}
