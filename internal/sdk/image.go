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
	io.Copy(os.Stdout, image.Body)

	_, err = apiclient.ImagePrune(ctx, client.ImagePruneOptions{})
	if err != nil {
		return err
	}

	slog.Info("Image built successfully", "image_name", imageName)
	return nil
}

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

func TagImage(ctx context.Context, apiclient *client.Client, source string, target string) error {
	// localhost:5000/source:latest
	imageResult, err := apiclient.ImageTag(ctx, client.ImageTagOptions{
		Source: source,
		Target: target,
	})

	if err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}

	fmt.Print("Image TAG Result", imageResult)
	return nil
}

func PushImage(ctx context.Context, apiclient *client.Client, target string) error {
	imagePush, err := apiclient.ImagePush(ctx, target, client.ImagePushOptions{})
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer imagePush.Close()
	io.Copy(os.Stdout, imagePush)
	return nil
}

func RemoveImage(ctx context.Context, apiclient *client.Client, target string) error {
	responses, err := apiclient.ImageRemove(ctx, target, client.ImageRemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	fmt.Print("Image Remove Result", responses)

	return nil
}
