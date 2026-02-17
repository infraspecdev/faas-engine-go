package sdk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

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
