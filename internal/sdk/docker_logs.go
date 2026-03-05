package sdk

import (
	"encoding/json"
	"faas-engine-go/internal/types"
	"fmt"
	"io"
	"strings"
)

var noisy = map[string]bool{
	"Pulling fs layer":  true,
	"Pull complete":     true,
	"Already exists":    true,
	"Download complete": true,
}

func streamDockerLogs(r io.Reader, out io.Writer) error {

	dec := json.NewDecoder(r)

	var prevLine string

	for {
		var msg types.DockerMessage

		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if msg.Error != "" {
			return fmt.Errorf("%s", msg.Error)
		}

		// Handle build stream output
		if msg.Stream != "" {
			line := strings.TrimSpace(msg.Stream)

			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "--->") && prevLine != "" {
				hash := strings.TrimSpace(strings.TrimPrefix(line, "--->"))
				fmt.Fprintf(out, "%s → %s\n", prevLine, hash)
				prevLine = ""
				continue
			}

			if strings.HasPrefix(line, "Step") {
				prevLine = line
				continue
			}

			fmt.Fprintln(out, line)
		}

		if msg.Status != "" {

			if noisy[msg.Status] {
				continue
			}

			if msg.ID != "" {

				if msg.Progress != "" {
					fmt.Fprintf(out, "%s: %s %s\n", msg.ID, msg.Status, msg.Progress)
				} else {
					fmt.Fprintf(out, "%s: %s\n", msg.ID, msg.Status)
				}

			} else {
				fmt.Fprintln(out, msg.Status)
			}
		}
	}
}
