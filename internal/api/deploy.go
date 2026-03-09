package api

import (
	"context"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Deployer interface {
	Deploy(ctx context.Context, name string, file io.Reader, out io.Writer) error
}

func DeployHandler(deployer Deployer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		w.Header().Set("Content-Type", "text/plain")

		r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadSize)

		if err := r.ParseMultipartForm(config.MaxUploadSize); err != nil {
			http.Error(w, "file too large", 400)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", 400)
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

		name := r.FormValue("name")

		out := &flushWriter{w, flusher}

		err = deployer.Deploy(r.Context(), name, file, out)
		if err != nil {
			fmt.Fprintf(out, "\nERROR: %s\n", err)
			return
		}

		fmt.Fprintf(out, "\nYour function is live at: http://%s.localhost\n", name)
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
