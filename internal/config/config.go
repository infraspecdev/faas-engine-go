package config

import (
	"fmt"
	"os"
)

// Registry returns the configured registry address for pushing function images.
// It checks the FAAS_REGISTRY environment variable and defaults to "localhost:5000" if not set.
func Registry() string {

	reg := os.Getenv("FAAS_REGISTRY")
	if reg == "" {
		return "localhost:5000"
	}
	return reg
}

// ImageRef constructs a full image reference for a function image based on the registry, namespace, name, and tag.
// If the tag is empty, it defaults to "latest".
func ImageRef(namespace, name, tag string) string {
	return fmt.Sprintf("%s/%s/%s:%s", Registry(), namespace, name, tag)
}

// RegistryUsername returns the configured registry username, if any.
func RegistryUsername() string {
	return os.Getenv("REGISTRY_USERNAME")
}

// RegistryPassword returns the configured registry password, if any.
func RegistryPassword() string {
	return os.Getenv("REGISTRY_PASSWORD")
}

// RuntimeURL returns the configured runtime manager URL.
// It checks the RUNTIME_URL environment variable and defaults to "http://localhost:8080" if not set.
func RuntimeURL() string {
	url := os.Getenv("RUNTIME_URL")
	if url == "" {
		return "http://localhost:8080"
	}
	return url
}

func ProxyURL() string {
	url := os.Getenv("PROXY_URL")
	if url == "" {
		return "http://localhost:8080"
	}
	return url
}
