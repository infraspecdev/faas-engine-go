package config

import "time"

// deploy function related constants
const (
	MaxUploadSize = 50 << 20
)

// invoke function related constants
const (
	ContainerPort                      = "8080/tcp"
	ContainerUser                      = "1000:1000"
	InitTimeout          time.Duration = 120 * time.Second
	PortTimeout          time.Duration = 10 * time.Second
	CleanUpTimeout       time.Duration = 12 * time.Second
	HealthTimeout        time.Duration = 10 * time.Second
	InvokeHTTPTimeout    time.Duration = 10 * time.Second
	ContainerStopTimeout time.Duration = 10 * time.Second
	ContainerIdleTimeout time.Duration = 10 * time.Second
)

// registry related constants
const (
	RegistryURL   = "localhost:5000"
	FunctionsRepo = "functions"
	RuntimesRepo  = "runtimes"
)
