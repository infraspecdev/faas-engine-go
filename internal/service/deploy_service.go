package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"
	"fmt"
	"io"
	"log/slog"

	"github.com/fatih/color"
)

type Deployer struct {
	imageClient sdk.ImageClient
	getVersion  func(name string) (string, error)
}

func NewDeployer(img sdk.ImageClient) *Deployer {
	return &Deployer{
		imageClient: img,
		getVersion: func(name string) (string, error) {
			return store.GetNextVersion(sqlite.DB, name)
		},
	}
}

// Deploy builds a Docker image from the provided file stream, tags it with the appropriate registry reference, pushes it to the registry, and then removes the local image.
// It logs each stage of the deployment process and returns an error if any step fails.
func (d *Deployer) Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error {
	logger := slog.With("function", name)

	cyan := color.New(color.FgCyan)
	_, _ = fmt.Fprint(out, "\n[2/3] Building image ")
	_, _ = cyan.Fprintf(out, "\"func-%s\"", name)
	_, _ = fmt.Fprint(out, "...\n\n")

	logger.Info("image_lifecycle", "stage", "building")
	if err := d.imageClient.BuildImage(ctx, name, file, out); err != nil {
		return err
	}

	version, err := d.getVersion(name)
	if err != nil {
		return err
	}

	target := config.ImageRef(config.FunctionsRepo, name, version)

	logger.Info("image_lifecycle", "stage", "tagging")
	if err := d.imageClient.TagImage(ctx, name, target); err != nil {
		return err
	}

	_, _ = fmt.Fprint(out, "...")
	if _, err := color.New(color.FgGreen).Fprintln(out, " Done."); err != nil {
		return err
	}

	_, _ = fmt.Fprint(out, "\n[3/3] Pushing image...")

	logger.Info("image_lifecycle", "stage", "pushing")
	if err := d.imageClient.PushImage(ctx, target); err != nil {
		return err
	}

	if _, err := color.New(color.FgGreen).Fprintln(out, " Done."); err != nil {
		return err
	}

	logger.Info("image_lifecycle", "stage", "removing_local")
	if err := d.imageClient.RemoveImage(ctx, name); err != nil {
		return err
	}

	logger.Info("image_lifecycle", "stage", "completed")
	return nil
}
