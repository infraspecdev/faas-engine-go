package service

import (
	"context"
	"faas-engine-go/internal/sdk"
	"io"

	"github.com/moby/moby/client"
)

type Deployer struct {
	CLI *client.Client
}

func (d *Deployer) Deploy(ctx context.Context, name string, file io.Reader) error {
	if err := sdk.CheckImageName(ctx, d.CLI, name); err != nil {
		return err
	}

	if err := sdk.BuildImage(ctx, d.CLI, name, file); err != nil {
		return err
	}

	target := "localhost:5000/functions/" + name

	if err := sdk.TagImage(ctx, d.CLI, name, target); err != nil {
		return err
	}

	if err := sdk.PushImage(ctx, d.CLI, target); err != nil {
		return err
	}

	if err := sdk.RemoveImage(ctx, d.CLI, name); err != nil {
		return err
	}

	return nil
}
