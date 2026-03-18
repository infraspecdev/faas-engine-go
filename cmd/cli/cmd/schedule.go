package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

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
	Long:  "Create a scheduled job that triggers a function based on a cron expression.",
	Example: `
  # Run function every 5 minutes
  faas schedule create calc --cron "*/5 * * * *"

  # Run function daily at midnight
  faas schedule create calc --cron "0 0 * * *"

  # Run with JSON payload
  faas schedule create calc --cron "*/10 * * * *" --data '{"a":10,"b":20}'
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := args[0]

		if cronExpr == "" {
			return fmt.Errorf("cron expression is required")
		}

		body := map[string]any{
			"cron": cronExpr,
		}

		if data != "" {
			var parsed any
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				return fmt.Errorf("invalid JSON payload: %w", err)
			}
			body["payload"] = parsed
		}

		reqBody, err := json.Marshal(body)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/functions/%s/schedule", serverAddr, functionName)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed: %s", resp.Status)
		}

		fmt.Println("schedule created successfully")
		return nil
	},
}

// -------------------- LIST --------------------

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled functions",
	Long:  "List all schedules or filter them by function name.",
	Example: `
  # List all schedules
  faas schedule list

  # List schedules for a specific function
  faas schedule list --function calc
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var url string

		if functionFilter != "" {
			url = fmt.Sprintf("%s/functions/%s/schedules", serverAddr, functionFilter)
		} else {
			url = fmt.Sprintf("%s/schedules", serverAddr)
		}

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed: %s", resp.Status)
		}

		var schedules []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
			return err
		}

		if len(schedules) == 0 {
			fmt.Println("No schedules found")
			return nil
		}

		for _, s := range schedules {
			fmt.Printf("ID: %v | Function: %v | Cron: %v\n",
				s["id"], s["function"], s["cron"])
		}

		return nil
	},
}

// -------------------- DELETE --------------------

var scheduleDeleteCmd = &cobra.Command{
	Use:   "delete [functionName] [scheduleID]",
	Short: "Delete a scheduled job",
	Long:  "Delete a specific schedule associated with a function.",
	Example: `
  # Delete a schedule
  faas schedule delete calc 123
`,
	Args: cobra.ExactArgs(2),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := args[0]
		id := args[1]

		req, err := http.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("%s/functions/%s/schedule/%s", serverAddr, functionName, id),
			nil,
		)
		if err != nil {
			return err
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed: %s", resp.Status)
		}

		fmt.Println("schedule deleted successfully")
		return nil
	},
}

// -------------------- INIT --------------------

func init() {
	// Flags for create
	scheduleCreateCmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	scheduleCreateCmd.Flags().StringVar(&data, "data", "", "JSON payload")

	// Flags for list
	scheduleListCmd.Flags().StringVarP(&functionFilter, "function", "f", "", "Filter by function name")

	// Attach subcommands
	scheduleCmd.AddCommand(scheduleCreateCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleDeleteCmd)

	// Attach to root
	rootCmd.AddCommand(scheduleCmd)
}
