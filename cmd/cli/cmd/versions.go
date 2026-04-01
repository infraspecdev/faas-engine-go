package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type VersionEntry struct {
	Version string `json:"version"`
	Active  bool   `json:"active"`
}

var versionsCmd = &cobra.Command{
	Use:   "versions <function-name>",
	Short: "list all versions of a function",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		functionName := strings.TrimSpace(args[0])
		if functionName == "" {
			return fmt.Errorf("function name is required")
		}

		endpoint := fmt.Sprintf(
			"%s/functions/%s/versions",
			serverAddr,
			url.PathEscape(functionName),
		)

		resp, err := http.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to fetch versions: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			color.Yellow("Function not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (%d): %s",
				resp.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}

		var versions []VersionEntry
		if err := json.Unmarshal(body, &versions); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(versions) == 0 {
			color.Yellow("No versions found.")
			return nil
		}

		color.Cyan("Versions:\n")

		for _, v := range versions {
			if v.Active {
				fmt.Printf("→ %s %s\n",
					color.GreenString(v.Version),
					"(active)",
				)
			} else {
				fmt.Printf("  %s\n", v.Version)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionsCmd)
}
