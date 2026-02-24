package buildcontext

import (
	"archive/tar"
	"encoding/json"
	"faas-engine-go/internal/api"
	"fmt"
	"io"
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

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		defer pw.Close()
		defer tw.Close()

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
			defer file.Close()

			_, err = io.Copy(tw, file)
			return err
		})

		if err != nil {
			pw.CloseWithError(err)
			return
		}
		dockerfile := "FROM localhost:5000/runtimes/node:v1\nCOPY . /function\n"

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
	}()

	return pr, nil
}

func SendTarStream(tarStream io.Reader, url string, functionName string) (string, error) {

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "function.tar")
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		_, err = io.Copy(part, tarStream)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		err = writer.WriteField("name", functionName)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response api.DeployResponse

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned %s", resp.Status)
	}

	return response.Message, err
}
