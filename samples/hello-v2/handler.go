package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	// Read input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeError("failed to read input")
		return
	}

	var event map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &event); err != nil {
			writeError("invalid JSON input")
			return
		}
	}

	// Extract name
	name, ok := event["name"].(string)
	if !ok || name == "" {
		name = "World"
	}

	// Create response
	response := map[string]string{
		"message": fmt.Sprintf("Hello, %s!", name),
	}

	// Write output to stdout
	json.NewEncoder(os.Stdout).Encode(response)
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(map[string]string{
		"error": msg,
	})
}
