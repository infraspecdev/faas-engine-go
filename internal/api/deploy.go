package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type Deployer interface {
	Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error
}

type FunctionStore interface {
	GetNextVersion(name string) (string, error)
	DeactivateFunctions(name string) error
	CreateFunction(fn *models.Function) error
}

type realFunctionStore struct {
	db *sql.DB
}

func NewFunctionStore(db *sql.DB) FunctionStore {
	return &realFunctionStore{db: db}
}

func (r *realFunctionStore) GetNextVersion(name string) (string, error) {
	return store.GetNextVersion(r.db, name)
}

func (r *realFunctionStore) DeactivateFunctions(name string) error {
	return store.DeactivateFunctions(r.db, name)
}

func (r *realFunctionStore) CreateFunction(fn *models.Function) error {
	return store.CreateFunction(r.db, fn)
}

// DeployHandler handles HTTP requests to deploy a function.
// It expects multipart form data containing the function package and the optional "name" field.
// The handler streams deployment progress and returns a deployment status trailer.
func DeployHandler(deployer Deployer, fs FunctionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Add("Trailer", "X-Deploy-Status")

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
		w.WriteHeader(http.StatusOK)

		err = deployer.Deploy(r.Context(), nameParam, file, out)
		if err != nil {
			_, _ = fmt.Fprintf(out, "\nERROR: %s\nSTREAM_STATUS: ERROR\n", err)
			// NOTE: Status code already sent (200 OK). Using trailer "X-Deploy-Status: ERROR"
			// to signal client that deployment failed. Client should parse final lines
			// to detect error status ("STREAM_STATUS: ERROR").
			w.Header().Set("X-Deploy-Status", "ERROR")
			return
		}

		functionName, functionVersion, found := strings.Cut(nameParam, ":")
		if !found {
			functionVersion, err = fs.GetNextVersion(functionName)
			if err != nil {
				slog.Error("failed to get latest version", "error", err)
				return
			}
		}

		if seeker, ok := file.(io.Seeker); ok {
			_, err = seeker.Seek(0, io.SeekStart)
			if err != nil {
				slog.Error("failed to reset file", "error", err)
				return
			}
		}

		checksum, err := calculateCheckSum(file)
		if err != nil {
			slog.Error("failed to calculate checksum", "error", err)
			return
		}

		err = fs.DeactivateFunctions(functionName)
		if err != nil {
			slog.Error("failed to deactivate old versions", "error", err)
		}

		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		var endpoint string
		if host == "localhost" || host == "127.0.0.1" {
			endpoint = fmt.Sprintf("%s.localhost", functionName)
		} else {
			endpoint = fmt.Sprintf("%s.%s.nip.io", functionName, host)
		}

		fn := &models.Function{
			Name:            functionName,
			Version:         functionVersion,
			PackageChecksum: checksum,
			Image:           config.ImageRef(config.FunctionsRepo, functionName, functionVersion),
			Runtime:         "node",
			ScheduleCron:    "",
			Endpoint:        endpoint,
			Status:          "active",
			CreatedAt:       time.Now(),
		}

		err = fs.CreateFunction(fn)
		if err != nil {
			slog.Error("failed to store function", "error", err)
			fmt.Fprintf(out, "\nWARNING: function deployed but DB insert failed\n")
		}

		_, _ = fmt.Fprintf(out, "\nYour function is live at: http://%s\n\n", endpoint)
		w.Header().Set("X-Deploy-Status", "OK")
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

	_, err := io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
