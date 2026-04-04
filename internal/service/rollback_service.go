package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"faas-engine-go/internal/sqlite/models"

	"github.com/google/uuid"
)

type ContainerClient interface {
	DeleteContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
}

type RollbackService struct {
	store            Store
	containerClient  ContainerClient
	lastRollback     map[string]time.Time
	mu               sync.Mutex
	rollbackCooldown time.Duration
}

func NewRollbackService(s Store, c ContainerClient) *RollbackService {
	return &RollbackService{
		store:            s,
		containerClient:  c,
		lastRollback:     make(map[string]time.Time),
		rollbackCooldown: 5 * time.Second,
	}
}

type RollbackResult struct {
	FunctionName    string    `json:"function_name"`
	PreviousVersion string    `json:"previous_version"`
	CurrentVersion  string    `json:"current_version"`
	RolledBackAt    time.Time `json:"rolled_back_at"`
	CleanupStatus   string    `json:"cleanup_status"`
	CleanupError    string    `json:"cleanup_error,omitempty"`
	RequestID       string    `json:"request_id"`
}

// Rollback reverts a function to a specific version.
// Rollback reverts a function to a specific version.
// Supports both explicit rollback (with targetVersion) and implicit rollback (without).
//
// DESIGN: All version selection logic is handled in RollbackToVersion() inside a DB transaction.
// This service layer only handles validation and async post-rollback cleanup.
func (rs *RollbackService) Rollback(ctx context.Context, functionName, targetVersion string) (*RollbackResult, error) {
	if functionName == "" {
		return nil, fmt.Errorf("function name is required")
	}

	rs.mu.Lock()
	lastTime := rs.lastRollback[functionName]
	rs.mu.Unlock()

	if time.Since(lastTime) < rs.rollbackCooldown {
		return nil, fmt.Errorf("rollback cooldown in effect; minimum %v between rollbacks", rs.rollbackCooldown)
	}

	// Validate explicit rollback target if provided
	if targetVersion != "" {
		err := rs.validateImageExists(ctx, functionName, targetVersion)
		if err != nil {
			return nil, fmt.Errorf("image validation failed: %w", err)
		}
	}

	requestID := uuid.New().String()

	// All rollback logic happens inside RollbackToVersionWithID transaction
	// This ensures atomicity and prevents race conditions
	// RollbackToVersionWithID returns the deactivatedFunctionID (the ID of the old version)
	// which we need for UpdateCleanupStatus to correctly track cleanup results
	previousVersion, deactivatedFunctionID, err := rs.store.RollbackToVersionWithID(functionName, targetVersion, requestID)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	// Get the new active version after rollback
	currentFunc, err := rs.store.GetActiveFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get active function after rollback: %w", err)
	}

	rs.mu.Lock()
	rs.lastRollback[functionName] = time.Now()
	rs.mu.Unlock()

	result := &RollbackResult{
		FunctionName:    functionName,
		PreviousVersion: previousVersion,
		CurrentVersion:  currentFunc.Version,
		RolledBackAt:    time.Now(),
		CleanupStatus:   "pending",
		RequestID:       requestID,
	}

	go func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cleanupErr := rs.cleanupOldContainers(cleanCtx, functionName, previousVersion)

		var status, errMsg string
		if cleanupErr != nil {
			status = "failed"
			errMsg = cleanupErr.Error()
			slog.Error("cleanup failed", "function", functionName, "error", cleanupErr)
		} else {
			status = "completed"
			errMsg = ""
			slog.Info("cleanup completed", "function", functionName, "version", previousVersion)
		}

		// Use deactivatedFunctionID (the ID of the old version from the transaction result)
		// NOT currentFunc.ID (the ID of the new active version)
		if err := rs.store.UpdateCleanupStatus(deactivatedFunctionID, requestID, status, errMsg); err != nil {
			slog.Error("failed to update cleanup status in database", "function", functionName, "request_id", requestID, "error", err)
		}

		// Update in-memory result (for any immediate queries on the result object)
		result.CleanupStatus = status
		result.CleanupError = errMsg
	}()

	slog.Info(
		"function_rollback",
		"function", functionName,
		"from", previousVersion,
		"to", targetVersion,
	)

	return result, nil
}

// cleanupOldContainers removes only FREE containers from the previous version
// BUSY containers are left alone to finish in-flight requests.
// Returns error if cleanup fails
// Note: This function is called asynchronously; errors do not block rollback completion.
// Cleanup status is tracked and returned in the RollbackResult.
func (rs *RollbackService) cleanupOldContainers(ctx context.Context, functionName, previousVersion string) error {
	fn, err := rs.store.ListFunctionVersions(functionName)
	if err != nil {
		return fmt.Errorf("cleanup: failed to fetch versions: %w", err)
	}

	var previousID int
	for _, f := range fn {
		if f.Version == previousVersion {
			previousID = f.ID
			break
		}
	}

	if previousID == 0 {
		return nil
	}

	containers, err := rs.store.GetContainersByFunction(previousID)
	if err != nil {
		return fmt.Errorf("cleanup: failed to fetch containers: %w", err)
	}

	var lastErr error
	for _, c := range containers {
		if c.Status == "free" {
			slog.Info("cleanup: removing free container", "container_id", c.ID, "version", previousVersion)

			if err := rs.containerClient.StopContainer(ctx, c.ID); err != nil {
				slog.Warn("cleanup: failed to stop container", "container_id", c.ID, "error", err)
				lastErr = err
			}

			if err := rs.containerClient.DeleteContainer(ctx, c.ID); err != nil {
				slog.Warn("cleanup: failed to delete container", "container_id", c.ID, "error", err)
				lastErr = err
			}

			if err := rs.store.RemoveContainer(c.ID); err != nil {
				slog.Warn("cleanup: failed to remove container from db", "container_id", c.ID, "error", err)
				lastErr = err
			}
		} else if c.Status == "busy" {
			slog.Info("cleanup: skipping busy container (will be cleaned by spleen)", "container_id", c.ID, "version", previousVersion)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("cleanup had errors: %w", lastErr)
	}
	return nil
}

// validateImageExists checks if target version image exists in DB .
func (rs *RollbackService) validateImageExists(ctx context.Context, functionName, targetVersion string) error {
	versions, err := rs.store.ListFunctionVersions(functionName)
	if err != nil {
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	if len(versions) == 0 {
		return fmt.Errorf("function not found")
	}

	var targetFunc *models.Function
	for i := range versions {
		if versions[i].Version == targetVersion {
			targetFunc = &versions[i]
			break
		}
	}

	if targetFunc == nil {
		return fmt.Errorf("target version %s not found", targetVersion)
	}

	if targetFunc.Image == "" {
		return fmt.Errorf("target version has no image registered")
	}

	slog.Debug("image validation passed", "function", functionName, "version", targetVersion, "image", targetFunc.Image)
	return nil
}

// GetRollbackHistory returns the rollback history for a function.
// Enforces max limit of 1000 to prevent DOS.
func (rs *RollbackService) GetRollbackHistory(functionName string, limit int) ([]map[string]interface{}, error) {
	const (
		defaultLimit = 20
		maxLimit     = 1000
	)

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
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
