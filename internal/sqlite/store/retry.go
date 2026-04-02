package store

import (
	"math/rand"
	"time"
)

// RetryWithBackoff retries a function with exponential backoff on busy errors
// Unified retry implementation used across all store operations
// Parameters:
//   - fn: Function to retry
//   - baseDelay: Initial delay (typically 50ms)
//   - maxRetries: Maximum number of retry attempts (typically 5)
func RetryWithBackoff(fn func() error, baseDelay time.Duration, maxRetries int) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()

		if err == nil {
			return nil
		}

		// Check if this is a retriable error (database is locked)
		if attempt < maxRetries-1 && isBusyError(err) {
			// Exponential backoff with jitter: 2^attempt * baseDelay + random
			// Example with 50ms base: 50ms, 100ms, 200ms, 400ms, then final attempt
			delay := baseDelay * time.Duration(1<<uint(attempt))
			// Add significant randomization (±50%) to prevent retry storms
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			time.Sleep(delay + jitter)
			continue
		}

		return err
	}

	return nil
}

// isBusyError checks if the error is a SQLite BUSY error
func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite returns error messages containing "database is locked"
	return err.Error() != "" && (contains(err.Error(), "database is locked") ||
		contains(err.Error(), "SQLITE_BUSY") ||
		contains(err.Error(), "disk I/O error"))
}

// contains is a simple string containment check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
