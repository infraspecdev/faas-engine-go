package buildcontext

import (
	"archive/tar"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func CreateTarStream(dirPath string) (io.Reader, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Check if Dockerfile already exists
	dockerfilePath := filepath.Join(dirPath, "Dockerfile")
	_, err = os.Stat(dockerfilePath)
	dockerfileExists := (err == nil)

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)

		defer func() {
			if err := pw.Close(); err != nil {
				log.Printf("failed to close pipe writer: %v", err)
			}
		}()

		defer func() {
			if err := tw.Close(); err != nil {
				log.Printf("failed to close tar writer: %v", err)
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
					log.Printf("failed to close file: %v", err)
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

			baseImage := config.ImageRef(config.RuntimesRepo, "node", "v1")

			dockerfile := fmt.Sprintf(
				"FROM %s\nCOPY . /function\n",
				baseImage,
			)

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

// func SendTarStream(tarStream io.Reader, url string, functionName string) (string, error) {

// 	pr, pw := io.Pipe()
// 	writer := multipart.NewWriter(pw)

// 	go func() {
// 		defer pw.Close()
// 		defer writer.Close()

// 		part, err := writer.CreateFormFile("file", "function.tar")
// 		if err != nil {
// 			pw.CloseWithError(err)
// 			return
// 		}

// 		_, err = io.Copy(part, tarStream)
// 		if err != nil {
// 			pw.CloseWithError(err)
// 			return
// 		}

// 		err = writer.WriteField("name", functionName)
// 		if err != nil {
// 			pw.CloseWithError(err)
// 			return
// 		}
// 	}()

// 	req, err := http.NewRequest("POST", url, pr)
// 	if err != nil {
// 		return "", err
// 	}

// 	req.Header.Set("Content-Type", writer.FormDataContentType())

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	var response types.DeployResponse

// 	if resp.StatusCode != http.StatusOK {
// 		body, _ := io.ReadAll(resp.Body)
// 		return "", fmt.Errorf("server returned %s: %s", resp.Status, string(body))
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(&response)
// 	if err != nil {
// 		return "", err
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return "", fmt.Errorf("server returned %s", resp.Status)
// 	}

// 	return response.Message, err
// }

func SendTarStream(tarStream io.Reader, url string, functionName string) error {

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func() {
			if err := pw.Close(); err != nil {
				log.Printf("failed to close pipe writer: %v", err)
			}
		}()

		defer func() {
			if err := writer.Close(); err != nil {
				log.Printf("failed to close tar writer: %v", err)
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
			log.Printf("failed to close response body: %v", err)
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
