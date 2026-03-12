package main

import (
	"faas-engine-go/cmd/cli/cmd"
	"log/slog"
	"os"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Execute root command
	cmd.Execute()
}
