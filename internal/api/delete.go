package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"

	"github.com/gorilla/mux"
)

type DeleteResponse struct {
	Message string   `json:"message"`
	Failed  []string `json:"failed_versions,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func retry(attempts int, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i+1) * config.RegistryDeleteTimeout)
	}

	return err
}

var errNotFound = errors.New("manifest not found")

func getDigest(name, version string) (string, error) {
	url := fmt.Sprintf("http://%s/v2/functions/%s/manifests/%s",
		config.Registry(),
		name,
		version,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

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

func deleteImage(name, digest string) error {
	url := fmt.Sprintf("http://%s/v2/functions/%s/manifests/%s",
		config.Registry(),
		name,
		digest,
	)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

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

func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["functionName"])

	if name == "" {
		http.Error(w, "functionName is required", http.StatusBadRequest)
		return
	}

	versions, err := store.ListFunctionVersions(sqlite.DB, name)
	if err != nil {
		http.Error(w, "failed to fetch function versions", http.StatusInternalServerError)
		return
	}

	if len(versions) == 0 {
		http.Error(w, "function not found", http.StatusNotFound)
		return
	}

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed []string
	)

	for _, v := range versions {
		version := v.Version

		wg.Add(1)

		go func() {
			defer wg.Done()

			digest, err := getDigest(name, version)

			if err != nil {
				if errors.Is(err, errNotFound) {
					slog.Info("already_deleted", "function", name, "version", version)
					return
				}

				slog.Error("digest_failed", "function", name, "version", version, "error", err)

				mu.Lock()
				failed = append(failed, version)
				mu.Unlock()
				return
			}

			if err := retry(config.RegistryDeleteRetries, func() error {
				return deleteImage(name, digest)
			}); err != nil {

				slog.Error("delete_failed", "function", name, "version", version, "error", err)

				mu.Lock()
				failed = append(failed, version)
				mu.Unlock()
				return
			}

			slog.Info("deleted_version", "function", name, "version", version)
		}()
	}

	wg.Wait()

	if len(failed) > 0 {
		writeJSON(w, http.StatusInternalServerError, DeleteResponse{
			Message: "failed to delete some versions",
			Failed:  failed,
		})
		return
	}

	if err := store.DeleteFunction(sqlite.DB, name); err != nil {
		http.Error(w, "failed to delete function from database", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, DeleteResponse{
		Message: "function deleted successfully",
	})
}
