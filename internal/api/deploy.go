package api

import (
	"context"
	"crypto/sha256"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Deployer interface {
	Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error
}

func DeployHandler(deployer Deployer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")

		r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)

		if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
			http.Error(w, "file too large", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				slog.Error("image_lifecycle",
					"stage", "failed_to_close_file",
					"error", err,
				)
			}
		}()

		nameParam := r.FormValue("name")

		out := &flushWriter{w, flusher}

		err = deployer.Deploy(r.Context(), nameParam, file, out)
		if err != nil {
			_, _ = fmt.Fprintf(out, "\nERROR: %s\n", err)
			return
		}

		functionName, functionVersion, found := strings.Cut(nameParam, ":")
		if !found {
			functionVersion, err = store.GetNextVersion(sqlite.DB, functionName)
			if err != nil {
				slog.Error("failed to get latest version", "error", err)
				return
			}
		}

		checksum, err := calculateCheckSum(file)
		if err != nil {
			slog.Error("failed to calculate checksum", "error", err)
			return
		}

		err = store.DeactivateFunctions(sqlite.DB, functionName)
		if err != nil {
			slog.Error("failed to deactivate old versions", "error", err)
		}

		fn := &models.Function{
			Name:            functionName,
			Version:         functionVersion,
			PackageChecksum: checksum,
			Image:           fmt.Sprintf("localhost:5000/functions/%s:%s", functionName, functionVersion),
			Runtime:         "node",
			ScheduleCron:    "",
			Endpoint:        fmt.Sprintf("%s.localhost", functionName),
			Status:          "active",
			CreatedAt:       time.Now(),
		}

		err = store.CreateFunction(sqlite.DB, fn)
		if err != nil {
			slog.Error("failed to store function", "error", err)
			fmt.Fprintf(out, "\nWARNING: function deployed but DB insert failed\n")
		}

		_, _ = fmt.Fprintf(out, "\nYour function is live at: http://%s.localhost\n", nameParam)
	}
}

type flushWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (f *flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	f.flusher.Flush()
	return n, err
}

func calculateCheckSum(file io.Reader) (string, error) {
	hasher := sha256.New()

	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return "", err
		}
	}

	_, err := io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
