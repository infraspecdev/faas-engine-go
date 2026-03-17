package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var cronExpr string

var scheduleCmd = &cobra.Command{
	Use:   "schedule [functionName]",
	Short: "Schedule a function using cron",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := args[0]

		if cronExpr == "" {
			fmt.Println("cron expression is required")
			return fmt.Errorf("cron expression is required")
		}

		// Prepare request body
		body := map[string]any{
			"cron": cronExpr,
		}

		// Add payload if provided
		if data != "" {
			var parsed any
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				fmt.Println("invalid JSON payload")
				return fmt.Errorf("invalid JSON payload: %w", err)
			}
			body["payload"] = parsed
		}

		reqBody, err := json.Marshal(body)
		if err != nil {
			fmt.Println("failed to create request body:", err)
			return fmt.Errorf("failed to create request body: %w", err)
		}

		url := fmt.Sprintf("%s/functions/%s/schedule", serverAddr, functionName)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Println("request failed:", err)
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			fmt.Println("failed:", resp.Status)
			return fmt.Errorf("failed: %s", resp.Status)
		}

		fmt.Println("schedule created successfully")
		return nil
	},
}

func init() {
	scheduleCmd.Flags().StringVar(&cronExpr, "cron", "", "cron expression")
	scheduleCmd.Flags().StringVar(&data, "data", "", "JSON payload")

	rootCmd.AddCommand(scheduleCmd)
}
