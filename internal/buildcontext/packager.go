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
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()

		filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				pw.CloseWithError(err)
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				pw.CloseWithError(err)
				return err
			}

			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				pw.CloseWithError(err)
				return err
			}

			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				pw.CloseWithError(err)
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					pw.CloseWithError(err)
					return err
				}
				defer file.Close()
				io.Copy(tw, file)
			}

			return nil
		})
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

	_, err = io.Copy(os.Stdout, resp.Body)

	return response.Message, err
}
