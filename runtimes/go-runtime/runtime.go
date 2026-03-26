package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const PORT = ":8080"
const EXEC_TIMEOUT = 5 * time.Second

var FUNCTION_FILE string

func init() {
	FUNCTION_FILE = os.Getenv("FUNCTION_FILE")
	if FUNCTION_FILE == "" {
		FUNCTION_FILE = "handler"
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", invokeHandler)

	log.Printf("Go runtime listening on %s\n", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}

//-------- Handlers --------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"error": "Method Not Allowed",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Failed to read body",
		})
		return
	}
	defer r.Body.Close()

	var event interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": fmt.Sprintf("Invalid JSON: %v", err),
			})
			return
		}
	} else {
		event = map[string]interface{}{}
	}

	result, err := executeFunction(event)
	if err != nil {
		log.Println("Function execution error:", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result": result,
	})
}

// -------- Core Execution --------

func executeFunction(event interface{}) (interface{}, error) {
	handlerPath := filepath.Join("/function", FUNCTION_FILE)

	info, err := os.Stat(handlerPath)
	if err != nil {
		return nil, fmt.Errorf("handler not found at %s: %w", handlerPath, err)
	}

	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("handler is not executable")
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), EXEC_TIMEOUT)
	defer cancel()

	cmd := exec.CommandContext(ctx, handlerPath)
	cmd.Stdin = bytes.NewReader(eventJSON)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	start := time.Now()
	output, err := cmd.Output()
	log.Printf("Execution time: %s\n", time.Since(start))

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("function execution timed out")
	}

	if err != nil {
		return nil, fmt.Errorf("handler failed: %v | stderr: %s", err, stderr.String())
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("handler returned empty output")
	}

	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("invalid handler output: %w", err)
	}

	return result, nil
}

// -------- Utils --------

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
