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
	// Container cleanup timeouts (for spleen/garbage collection)
	ContainerCleanupStopTimeout   time.Duration = 30 * time.Second
	ContainerCleanupDeleteTimeout time.Duration = 30 * time.Second
	// Log retrieval timeout after invocation
	LogRetrievalTimeout time.Duration = 2 * time.Second
	// CLI invoke timeout
	CLIInvokeTimeout time.Duration = 15 * time.Second
)

// delete function related constants
const (
	RegistryDeleteTimeout = 2 * time.Second
	RegistryDeleteRetries = 3
)

// schedule related constants
const (
	ScheduleMinimumIntervalSeconds time.Duration = 60 * time.Second
)

// graceful shutdown related constants
const (
	ServerShutdownTimeout       time.Duration = 30 * time.Second
	GracefulShutdownTimeout     time.Duration = 120 * time.Second
	ShutdownContainerStopTime   time.Duration = 20 * time.Second
	ShutdownContainerDeleteTime time.Duration = 15 * time.Second
)

// registry related constants
const (
	RegistryURL   = "localhost:5000"
	FunctionsRepo = "functions"
	RuntimesRepo  = "runtimes"
)

// database related constants
const (
	// SQLite busy timeout in milliseconds (10 seconds)
	SQLiteBusyTimeout = 10000
	// Max open connections for database
	MaxOpenConns = 25
	// Max idle connections for database
	MaxIdleConns = 5
	// Database queue operation timeout
	DBQueueTimeout time.Duration = 30 * time.Second
)

// server related constants
const (
	DefaultServerAddr = "http://localhost:8080"
)
