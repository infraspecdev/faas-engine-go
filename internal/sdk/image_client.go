package sdk

import (
	"context"
	"io"
)

// ImageClient defines the Docker image operations used by the FaaS engine.
type ImageClient interface {
	PullImage(ctx context.Context, imageName string) error
	BuildImage(ctx context.Context, imageName string, tarfile io.Reader) error
	TagImage(ctx context.Context, source string, target string) error
	PushImage(ctx context.Context, target string) error
	RemoveImage(ctx context.Context, target string) error
}
