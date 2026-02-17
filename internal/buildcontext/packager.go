package buildcontext

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func PackageFunction(dirPath string) (io.Reader, error) {

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
