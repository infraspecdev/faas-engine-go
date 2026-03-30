package store

import (
	"math/rand"
	"time"
)

// retryWithBackoff retries a function with exponential backoff on busy errors
func retryWithBackoff(fn func() error) error {
	const maxRetries = 5 // Allow up to 5 retry attempts
	const baseDelay = 50 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()

		if err == nil {
			return nil
		}

		// Check if this is a retriable error (database is locked)
		if attempt < maxRetries-1 && isBusyError(err) {
			// Exponential backoff with jitter: 50ms * 2^attempt + random
			// Attempts: 50ms, 100ms, 200ms, 400ms, then final attempt
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
