package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type ContainerClient interface {
	DeleteContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
}

type RollbackService struct {
	store           Store
	containerClient ContainerClient
}

func NewRollbackService(s Store, c ContainerClient) *RollbackService {
	return &RollbackService{
		store:           s,
		containerClient: c,
	}
}

type RollbackResult struct {
	FunctionName    string    `json:"function_name"`
	PreviousVersion string    `json:"previous_version"`
	CurrentVersion  string    `json:"current_version"`
	RolledBackAt    time.Time `json:"rolled_back_at"`
}

// Rollback reverts a function to a specific version.
// If targetVersion is empty, rolls back to the previous version.
func (rs *RollbackService) Rollback(ctx context.Context, functionName, targetVersion string) (*RollbackResult, error) {
	if functionName == "" {
		return nil, fmt.Errorf("function name is required")
	}

	// Get all versions to find the actual target
	versions, err := rs.store.ListFunctionVersions(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("function not found")
	}

	// If no target specified, rollback to previous version
	if targetVersion == "" {
		// Find the active version
		var activeIdx int
		for i, v := range versions {
			if v.Status == "active" {
				activeIdx = i
				break
			}
		}

		if activeIdx >= len(versions)-1 {
			return nil, fmt.Errorf("no previous version to rollback to")
		}

		// Previous version is the next in the DESC ordered list
		targetVersion = versions[activeIdx+1].Version
	} else {
		// Validate target version exists
		found := false
		for _, v := range versions {
			if v.Version == targetVersion {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("target version %s not found", targetVersion)
		}
	}

	// Perform atomic rollback
	previousVersion, err := rs.store.RollbackToVersion(functionName, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	// Cleanup old containers asynchronously
	go func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rs.cleanupOldContainers(cleanCtx, functionName, previousVersion)
	}()

	slog.Info(
		"function_rollback",
		"function", functionName,
		"from", previousVersion,
		"to", targetVersion,
	)

	return &RollbackResult{
		FunctionName:    functionName,
		PreviousVersion: previousVersion,
		CurrentVersion:  targetVersion,
		RolledBackAt:    time.Now(),
	}, nil
}

// cleanupOldContainers stops and removes containers from the previous version
func (rs *RollbackService) cleanupOldContainers(ctx context.Context, functionName, previousVersion string) {
	fn, err := rs.store.ListFunctionVersions(functionName)
	if err != nil {
		slog.Error("cleanup: failed to fetch versions", "error", err)
		return
	}

	var previousID int
	for _, f := range fn {
		if f.Version == previousVersion {
			previousID = f.ID
			break
		}
	}

	if previousID == 0 {
		return
	}

	containers, err := rs.store.GetContainersByFunction(previousID)
	if err != nil {
		slog.Error("cleanup: failed to fetch containers", "error", err)
		return
	}

	for _, c := range containers {
		if c.Status == "busy" || c.Status == "free" {
			slog.Info("cleanup: removing container", "container_id", c.ID, "version", previousVersion)

			if err := rs.containerClient.StopContainer(ctx, c.ID); err != nil {
				slog.Warn("cleanup: failed to stop container", "container_id", c.ID, "error", err)
			}

			if err := rs.containerClient.DeleteContainer(ctx, c.ID); err != nil {
				slog.Warn("cleanup: failed to delete container", "container_id", c.ID, "error", err)
			}

			if err := rs.store.RemoveContainer(c.ID); err != nil {
				slog.Warn("cleanup: failed to remove container from db", "container_id", c.ID, "error", err)
			}
		}
	}
}

// GetRollbackHistory returns the rollback history for a function
func (rs *RollbackService) GetRollbackHistory(functionName string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}

	history, err := rs.store.GetVersionHistory(functionName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	var result []map[string]interface{}
	for _, h := range history {
		result = append(result, map[string]interface{}{
			"from": h.FromVersion,
			"to":   h.ToVersion,
			"at":   h.TriggeredAt,
		})
	}

	return result, nil
}
