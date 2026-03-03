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
	if tag == "" {
		tag = "latest"
	}
	return fmt.Sprintf("%s/%s/%s:%s", Registry(), namespace, name, tag)
}
