package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"fmt"
	"io"
	"log/slog"

	"github.com/moby/moby/client"
)

type Deployer struct {
	CLI *client.Client
}

// Deploy builds a Docker image from the provided file stream, tags it with the appropriate registry reference, pushes it to the registry, and then removes the local image.
// It logs each stage of the deployment process and returns an error if any step fails.
func (d *Deployer) Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error {
	logger := slog.With("function", name)

	fmt.Fprint(out, "\n[2/3] Building function image...\n\n")

	logger.Info("image_lifecycle", "stage", "building")
	if err := sdk.BuildImage(ctx, d.CLI, name, file, out); err != nil {
		return err
	}

	target := config.ImageRef(config.FunctionsRepo, name, "")

	logger.Info("image_lifecycle", "stage", "tagging")
	if err := sdk.TagImage(ctx, d.CLI, name, target); err != nil {
		return err
	}

	fmt.Fprint(out, "\n[3/3] Pushing image...")

	logger.Info("image_lifecycle", "stage", "pushing")
	if err := sdk.PushImage(ctx, d.CLI, target); err != nil {
		return err
	}

	fmt.Fprintln(out, " Done.")

	logger.Info("image_lifecycle", "stage", "removing_local")
	if err := sdk.RemoveImage(ctx, d.CLI, name); err != nil {
		return err
	}

	logger.Info("image_lifecycle", "stage", "completed")
	return nil
}
