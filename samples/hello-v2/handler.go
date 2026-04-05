package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
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

	name, ok := event["name"].(string)
	if !ok || name == "" {
		name = "World"
	}

	response := map[string]string{
		"message": fmt.Sprintf("Hello, %s!", name),
	}

	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		// Last resort: log to stderr (stdout is already compromised)
		fmt.Fprintln(os.Stderr, "failed to write response:", err)
	}
}

func writeError(msg string) {
	if err := json.NewEncoder(os.Stdout).Encode(map[string]string{
		"error": msg,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "failed to write error:", err)
	}
}
