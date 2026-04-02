package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type Schedule struct {
	ID           string `json:"id"`
	Functionname string `json:"function"`
	Cron         string `json:"cron"`
}

var cronExpr string
var functionFilter string

// Parent command
var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage function schedules",
	Long:  "Create, list, and delete scheduled executions of functions using cron expressions.",
}

// -------------------- CREATE --------------------

var scheduleCreateCmd = &cobra.Command{
	Use:   "create [functionName]",
	Short: "Schedule a function using a cron expression",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := args[0]

		if cronExpr == "" {
			color.Red("✗ Cron expression is required")
			return fmt.Errorf("use --cron flag to specify a cron expression")
		}

		body := map[string]any{
			"cron": cronExpr,
		}

		if data != "" {
			var parsed any
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				color.Red("✗ Invalid JSON payload")
				return fmt.Errorf("failed to parse JSON payload: %w", err)
			}
			body["payload"] = parsed
		}

		reqBody, err := json.Marshal(body)
		if err != nil {
			color.Red("✗ Failed to create schedule")
			return err
		}

		// ✅ FIXED ENDPOINT
		url := fmt.Sprintf("%s/schedules/%s", serverAddr, functionName)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			color.Red("✗ Failed to connect to server")
			return fmt.Errorf("unable to reach runtime manager at %s: %w", serverAddr, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			// Parse error response to get detailed message
			var errResp map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
				if errMsg, ok := errResp["error"]; ok {
					color.Red("✗ Failed to create schedule")
					return fmt.Errorf("%s", errMsg)
				}
			}
			color.Red("✗ Failed to create schedule")
			return fmt.Errorf("server error: %s", resp.Status)
		}

		color.Green("✅ Schedule created successfully for %s", functionName)
		return nil
	},
}

// -------------------- LIST --------------------

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled functions",

	RunE: func(cmd *cobra.Command, args []string) error {

		url := fmt.Sprintf("%s/schedules", serverAddr)

		// Add optional query parameter for function filtering (server-side filtering)
		if functionFilter != "" {
			url = fmt.Sprintf("%s?function=%s", url, functionFilter)
		}

		resp, err := http.Get(url)
		if err != nil {
			color.Red("✗ Failed to connect to server")
			return fmt.Errorf("unable to reach runtime manager at %s: %w", serverAddr, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			color.Red("✗ Failed to list schedules")
			return fmt.Errorf("server error: %s", resp.Status)
		}

		var schedules []Schedule
		if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
			color.Red("✗ Failed to parse response")
			return fmt.Errorf("invalid response format: %w", err)
		}

		if len(schedules) == 0 {
			color.Yellow("℧ No schedules found")
			return nil
		}

		header := color.New(color.FgCyan, color.Bold)
		idCol := color.New(color.FgGreen)
		fnCol := color.New(color.FgGreen, color.Bold)
		cronCol := color.New(color.FgWhite)
		border := color.New(color.FgHiBlack)

		// Header
		fmt.Println()
		header.Printf("%-37s %-15s %-20s\n", "ID", "FUNCTION", "CRON")
		border.Println("----------------------------------------------------------")

		for _, s := range schedules {

			idCol.Printf("%-37s ", s.ID)
			fnCol.Printf("%-15s ", s.Functionname)
			cronCol.Printf("%-20s\n", s.Cron)
		}

		fmt.Println()
		return nil
	},
}

// -------------------- DELETE --------------------

var scheduleDeleteCmd = &cobra.Command{
	Use:   "delete [scheduleID]",
	Short: "Delete a scheduled job",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		req, err := http.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("%s/schedules/%s", serverAddr, id),
			nil,
		)
		if err != nil {
			color.Red("✗ Failed to delete schedule")
			return err
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			color.Red("✗ Failed to connect to server")
			return fmt.Errorf("unable to reach runtime manager at %s: %w", serverAddr, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			color.Yellow("℧ Schedule not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			// Parse error response to get detailed message
			var errResp map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
				if errMsg, ok := errResp["error"]; ok {
					color.Red("✗ Failed to delete schedule")
					return fmt.Errorf("%s", errMsg)
				}
			}
			color.Red("✗ Failed to delete schedule")
			return fmt.Errorf("server error: %s", resp.Status)
		}

		color.Green("✅ Schedule deleted successfully")
		return nil
	},
}

// -------------------- INIT --------------------

func init() {
	// Flags
	scheduleCreateCmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	scheduleCreateCmd.Flags().StringVar(&data, "data", "", "JSON payload")

	scheduleListCmd.Flags().StringVarP(&functionFilter, "function", "f", "", "Filter by function name")

	// Attach subcommands
	scheduleCmd.AddCommand(scheduleCreateCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleDeleteCmd)

	rootCmd.AddCommand(scheduleCmd)
}
