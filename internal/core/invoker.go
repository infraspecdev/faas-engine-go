package core

import "context"

type Invoker interface {
	Invoke(ctx context.Context, functionName string, payload []byte, triggerType string) (any, error)
}
