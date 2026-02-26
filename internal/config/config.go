package config

import "os"

func Registry() string {
	reg := os.Getenv("FASS_REGISTRY")
	if reg == "" {
		return "localhost:5000"
	}
	return reg
}

func FunctionsNamespace() string {
	ns := os.Getenv("FASS_NAMESPACE")
	if ns == "" {
		return "functions"
	}
	return ns
}
