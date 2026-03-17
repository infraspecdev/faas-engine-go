package db

import (
	"encoding/json"
	"sync"
	"time"

	"faas-engine-go/internal/types"

	"github.com/google/uuid"
)

var (
	invocationStore = make(map[string]types.Invocation)
	invocationMu    sync.Mutex
)

// CREATE
func CreateInvocation(functionName string, payload []byte, trigger string) string {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	id := uuid.New().String()

	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		parsed = string(payload) // fallback
	}

	invocationStore[id] = types.Invocation{
		ID:             id,
		FunctionName:   functionName,
		TriggerType:    trigger,
		Status:         "running",
		RequestPayload: parsed,
		StartedAt:      time.Now(),
	}

	return id
}

// SET CONTAINER ID
func SetContainerID(id string, containerID string) {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	inv, ok := invocationStore[id]
	if !ok {
		return
	}

	inv.ContainerID = containerID
	invocationStore[id] = inv
}

// COMPLETE (SUCCESS)
func CompleteInvocation(id string, response any, duration time.Duration) {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	inv, ok := invocationStore[id]
	if !ok {
		return
	}

	inv.Status = "success"
	inv.ResponsePayload = response
	inv.DurationMs = duration.Milliseconds()
	inv.FinishedAt = time.Now()

	invocationStore[id] = inv
}

// FAIL
func FailInvocation(id string, errMsg string, duration time.Duration) {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	inv, ok := invocationStore[id]
	if !ok {
		return
	}

	inv.Status = "failed"
	inv.ResponsePayload = errMsg
	inv.DurationMs = duration.Milliseconds()
	inv.FinishedAt = time.Now()

	invocationStore[id] = inv
}

// READ
func GetInvocation(id string) (types.Invocation, bool) {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	inv, ok := invocationStore[id]
	return inv, ok
}

func ListInvocations() []types.Invocation {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	var result []types.Invocation
	for _, inv := range invocationStore {
		result = append(result, inv)
	}
	return result
}

func GetInvocationsByFunction(fn string) []types.Invocation {
	invocationMu.Lock()
	defer invocationMu.Unlock()

	var result []types.Invocation
	for _, inv := range invocationStore {
		if inv.FunctionName == fn {
			result = append(result, inv)
		}
	}
	return result
}
