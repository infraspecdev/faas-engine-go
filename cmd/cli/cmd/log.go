package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type LogEntry struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Logs     string `json:"logs"`
	Duration int    `json:"duration"`
}

var (
	version string
	limit   int
)

var logsCmd = &cobra.Command{
	Use:   "logs <function-name>",
	Short: "fetch logs for a function",
	Long:  `Fetch logs for the active version or a specific version of a function.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		functionName := strings.TrimSpace(args[0])
		if functionName == "" {
			return fmt.Errorf("function name is required")
		}

		endpoint := fmt.Sprintf("%s/functions/%s/log",
			serverAddr,
			url.PathEscape(functionName),
		)

		query := url.Values{}
		if strings.TrimSpace(version) != "" {
			query.Set("version", strings.TrimSpace(version))
		}
		if limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", limit))
		}

		if encoded := query.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}

		resp, err := http.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to fetch logs: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (%d): %s",
				resp.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}

		var logs []LogEntry
		if err := json.Unmarshal(body, &logs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(logs) == 0 {
			color.Yellow("No logs found.")
			return nil
		}

		for _, entry := range logs {

			status := entry.Status
			switch strings.ToLower(status) {
			case "success":
				status = color.GreenString(status)
			case "failed":
				status = color.RedString(status)
			case "running":
				status = color.CyanString(status)
			default:
				status = color.YellowString(status)
			}

			duration := time.Duration(entry.Duration) * time.Millisecond

			fmt.Println("--------------------------")
			fmt.Printf("Invocation: %s (%s) ~ (%s)\n", entry.ID, status, duration)
			fmt.Println("--------------------------")

			logText := strings.TrimSpace(entry.Logs)
			if logText == "" {
				color.Yellow("No logs available")
			} else {
				fmt.Println(logText)
			}

			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringVar(&version, "version", "", "Function version (optional)")
	logsCmd.Flags().IntVar(&limit, "limit", 20, "Number of logs to fetch")
}
