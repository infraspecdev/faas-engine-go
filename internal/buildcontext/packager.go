package buildcontext

import (
	"archive/tar"
	"encoding/json"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ValidateFunction(runtime string, dirPath string) error {
	switch runtime {

	case "node":
		return validateNode(dirPath)

	case "python":
		return validatePython(dirPath)

	case "go":
		return validateGo(dirPath)

	default:
		return fmt.Errorf("unsupported runtime %q (supported: node, python, go)\n", runtime)
	}
}

func validateNode(dir string) error {
	entryFile, err := findNodeEntryPoint(dir)
	if err != nil {
		return err
	}

	if entryFile == "" {
		return nil
	}

	cmd := exec.Command("node", "--check", entryFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error in %s:\n%s", filepath.Base(entryFile), string(out))
	}

	return nil
}

func findNodeEntryPoint(dir string) (string, error) {
	indexPath := filepath.Join(dir, "index.js")
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath, nil
	}

	packagePath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var pkg struct {
		Main string `json:"main"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}

	if pkg.Main == "" {
		return "", nil
	}

	return filepath.Join(dir, pkg.Main), nil
}

func validatePython(dir string) error {
	cmd := exec.Command("python", "-m", "compileall", dir)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python syntax error:\n%s", string(out))
	}

	return nil
}

func validateGo(dir string) error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build error:\n%s", string(out))
	}

	return nil
}

func CreateTarStream(dirPath string, runtime string) (io.Reader, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	err = ValidateFunction(runtime, dirPath)
	if err != nil {
		return nil, err
	}

	// Check if Dockerfile already exists
	dockerfilePath := filepath.Join(dirPath, "Dockerfile")
	_, err = os.Stat(dockerfilePath)
	dockerfileExists := (err == nil)

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)

		defer func() {
			if err := tw.Close(); err != nil {
				// Ignore "read/write on closed pipe" errors - this happens when Docker closes the connection
				if !strings.Contains(err.Error(), "closed pipe") {
					slog.Error("failed to close tar writer", "error", err)
				}
			}
		}()

		defer func() {
			if err := pw.Close(); err != nil {
				// Ignore "read/write on closed pipe" errors - this happens when Docker closes the connection
				if !strings.Contains(err.Error(), "closed pipe") {
					slog.Error("failed to close pipe writer", "error", err)
				}
			}
		}()

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}

			if relPath == "." {
				return nil
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				if err := file.Close(); err != nil {
					slog.Error("failed to close file", "path", path, "error", err)
				}
			}()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed copying %s: %w", relPath, err)
			}
			return nil
		})

		if err != nil {
			pw.CloseWithError(err)
			return
		}

		// Inject Dockerfile only if not present
		if !dockerfileExists {
			// slog.Info("No Dockerfile found, injecting default Dockerfile into build context")

			var dockerfile string

			switch runtime {

			case "node":
				baseImage := config.ImageRef(config.RuntimesRepo, "node", "v1")
				dockerfile = fmt.Sprintf(
					"FROM %s\nCOPY . /function\n",
					baseImage,
				)

			case "python":
				baseImage := config.ImageRef(config.RuntimesRepo, "python", "v1")
				dockerfile = fmt.Sprintf(
					"FROM %s\nCOPY . /function\n",
					baseImage,
				)

			case "go":
				baseImage := config.ImageRef(config.RuntimesRepo, "go", "v1")
				dockerfile = fmt.Sprintf(`
				FROM golang:1.22-alpine AS builder
				WORKDIR /build
				COPY . .
				RUN go build -o handler handler.go

				FROM %s
				COPY --from=builder /build/handler /function/handler
				RUN chmod +x /function/handler
				`, baseImage)

			default:
				pw.CloseWithError(fmt.Errorf("unsupported runtime: %s\nUse node, python or go", runtime))
				return
			}

			// slog.Info("Using registry", "value", config.Registry())

			dfBytes := []byte(dockerfile)

			header := &tar.Header{
				Name: "Dockerfile",
				Mode: 0644,
				Size: int64(len(dfBytes)),
			}

			if err := tw.WriteHeader(header); err != nil {
				pw.CloseWithError(err)
				return
			}

			if _, err := tw.Write(dfBytes); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr, nil
}

func SendTarStream(tarStream io.Reader, url string, functionName string) error {

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func() {
			if err := writer.Close(); err != nil {
				// Ignore "read/write on closed pipe" errors
				if !strings.Contains(err.Error(), "closed pipe") {
					slog.Error("failed to close multipart writer", "error", err)
				}
			}
		}()

		defer func() {
			if err := pw.Close(); err != nil {
				// Ignore "read/write on closed pipe" errors
				if !strings.Contains(err.Error(), "closed pipe") {
					slog.Error("failed to close pipe writer", "error", err)
				}
			}
		}()

		part, err := writer.CreateFormFile("file", "function.tar")
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		if _, err := io.Copy(part, tarStream); err != nil {
			pw.CloseWithError(err)
			return
		}

		if err := writer.WriteField("name", functionName); err != nil {
			pw.CloseWithError(err)
			return
		}

		if err := writer.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}

		if err := pw.Close(); err != nil {
			// Ignore "read/write on closed pipe" errors
			if !strings.Contains(err.Error(), "closed pipe") {
				slog.Error("failed to close pipe writer", "error", err)
			}
		}
	}()

	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %s: %s", resp.Status, string(body))
	}

	// STREAM SERVER OUTPUT
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// CheckFunctionExists checks if a function with the given name already exists
// by querying the list of all functions
// Returns true if function exists, false otherwise, and any error that occurred
func CheckFunctionExists(serverAddr string, functionName string) (bool, error) {
	url := fmt.Sprintf("%s/functions", serverAddr)

	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to check function: %w", err)
	}
	defer resp.Body.Close()

	// If we can't get the list, assume we can't verify
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("server returned %s: %s", resp.Status, string(body))
	}

	// Parse the response to look for our function
	var response struct {
		Functions []struct {
			Name string `json:"name"`
		} `json:"functions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("failed to parse function list: %w", err)
	}

	// Check if our function is in the list
	for _, fn := range response.Functions {
		if fn.Name == functionName {
			return true, nil
		}
	}

	return false, nil
}
