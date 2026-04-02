package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sqlite/models"
)

type DeleteStore interface {
	ListFunctionVersions(name string) ([]models.Function, error)
	DeleteFunction(name string) error
}

type RegistryClient interface {
	GetDigest(name, version string) (string, error)
	DeleteImage(name, digest string) error
}

type RetryFunc func(attempts int, fn func() error) error

var (
	ErrFunctionNotFound = errors.New("function not found")
	errNotFound         = errors.New("manifest not found")
)

type functionDeleteService struct {
	store    DeleteStore
	registry RegistryClient
	retry    RetryFunc
}

func NewFunctionDeleteService(s DeleteStore, r RegistryClient) *functionDeleteService {
	return &functionDeleteService{
		store:    s,
		registry: r,
		retry:    defaultRetry,
	}
}

func (s *functionDeleteService) DeleteFunction(name string) ([]string, error) {

	versions, err := s.store.ListFunctionVersions(name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch function versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, ErrFunctionNotFound
	}

	var (
		wg         sync.WaitGroup
		failedList []string
		mu         sync.Mutex
	)

	for _, v := range versions {
		version := v.Version

		wg.Add(1)

		go func(ver string) {
			defer wg.Done()

			digest, err := s.registry.GetDigest(name, ver)
			if err != nil {
				if errors.Is(err, errNotFound) {
					slog.Info("already_deleted", "function", name, "version", ver)
					return
				}

				// If registry is unreachable or connection error, skip deletion from registry
				// This allows deleting functions that are only deployed locally
				slog.Warn("registry_unavailable_skipping_image_delete", "function", name, "version", ver, "error", err.Error())
				return
			}

			err = s.retry(config.RegistryDeleteRetries, func() error {
				return s.registry.DeleteImage(name, digest)
			})

			if err != nil {
				// Check if it's a 405 Method Not Allowed error (registry doesn't support DELETE)
				if err.Error() == "delete failed: 405 Method Not Allowed" {
					slog.Info("registry_delete_not_supported", "function", name, "version", ver, "note", "registry does not support DELETE operations")
					return
				}
				slog.Warn("registry_image_delete_failed", "function", name, "version", ver, "error", err.Error())
				// Track failed versions but continue with database deletion
				mu.Lock()
				failedList = append(failedList, ver)
				mu.Unlock()
				return
			}
		}(version)
	}

	wg.Wait()

	if err := s.store.DeleteFunction(name); err != nil {
		return nil, fmt.Errorf("failed to delete function from db: %w", err)
	}

	// Return failed versions if any, but still consider it a successful deletion if database was cleaned
	if len(failedList) > 0 {
		return failedList, nil
	}

	return nil, nil
}

func defaultRetry(attempts int, fn func() error) error {
	var err error
	baseBackoff := 100 * time.Millisecond // Start with 100ms backoff

	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			// Exponential backoff: 100ms, 200ms, 400ms, etc.
			backoff := baseBackoff * time.Duration(1<<uint(i))
			time.Sleep(backoff)
		}
	}

	return err
}

type HTTPRegistryClient struct{}

func (c *HTTPRegistryClient) GetDigest(name, version string) (string, error) {
	url := fmt.Sprintf("http://%s/v2/functions/%s/manifests/%s",
		config.Registry(), name, version)

	req, _ := http.NewRequest(http.MethodGet, url, nil)

	req.Header.Set("Accept",
		"application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json",
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		digest := resp.Header.Get("Docker-Content-Digest")
		if digest == "" {
			return "", fmt.Errorf("digest not found")
		}
		return digest, nil

	case http.StatusNotFound:
		return "", errNotFound

	default:
		return "", fmt.Errorf("manifest fetch failed: %s", resp.Status)
	}
}

func (c *HTTPRegistryClient) DeleteImage(name, digest string) error {
	url := fmt.Sprintf("http://%s/v2/functions/%s/manifests/%s",
		config.Registry(), name, digest)

	req, _ := http.NewRequest(http.MethodDelete, url, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("delete failed: %s", resp.Status)
	}

	return nil
}
