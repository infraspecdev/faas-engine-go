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

type FunctionDeleter interface {
	DeleteFunction(name string) ([]string, error)
}

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

func NewFunctionDeleteService(s DeleteStore, r RegistryClient) FunctionDeleter {
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
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed []string
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

				mu.Lock()
				failed = append(failed, ver)
				mu.Unlock()
				return
			}

			err = s.retry(config.RegistryDeleteRetries, func() error {
				return s.registry.DeleteImage(name, digest)
			})

			if err != nil {
				mu.Lock()
				failed = append(failed, ver)
				mu.Unlock()
			}
		}(version)
	}

	wg.Wait()

	if len(failed) > 0 {
		return failed, fmt.Errorf("partial delete failure")
	}

	if err := s.store.DeleteFunction(name); err != nil {
		return nil, fmt.Errorf("failed to delete function from db: %w", err)
	}

	return nil, nil
}

func defaultRetry(attempts int, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i+1) * config.RegistryDeleteTimeout)
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
