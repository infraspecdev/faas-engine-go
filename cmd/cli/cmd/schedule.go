package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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

var (
	deleteAllFlag       bool
	deleteRemoveAllFlag bool
)

var scheduleDeleteCmd = &cobra.Command{
	Use:   "delete [scheduleID or functionName]",
	Short: "Delete scheduled jobs",
	Long:  `Delete schedules by ID, by function name with --all flag, or remove all schedules with --removeall`,
	Args:  cobra.MaximumNArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		// Check conflicting flags
		if deleteRemoveAllFlag && len(args) > 0 {
			color.Red("✗ Cannot specify both --removeall and a schedule ID or function name")
			return fmt.Errorf("conflicting flags")
		}

		if deleteRemoveAllFlag && deleteAllFlag {
			color.Red("✗ Cannot use --removeall and --all together")
			return fmt.Errorf("conflicting flags")
		}

		// Case 1: Delete all schedules globally with --removeall
		if deleteRemoveAllFlag {
			if !confirmAction("Are you sure you want to delete ALL schedules?") {
				color.Yellow("✗ Operation cancelled")
				return nil
			}

			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/schedules?removeall=true", serverAddr),
				nil,
			)
			if err != nil {
				color.Red("✗ Failed to delete schedules")
				return err
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				color.Red("✗ Failed to connect to server")
				return fmt.Errorf("unable to reach runtime manager at %s: %w", serverAddr, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				var errResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
					if errMsg, ok := errResp["error"]; ok {
						color.Red("✗ Failed to delete schedules")
						return fmt.Errorf("%s", errMsg)
					}
				}
				color.Red("✗ Failed to delete schedules")
				return fmt.Errorf("server error: %s", resp.Status)
			}

			var resp_body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&resp_body); err != nil {
				color.Green("✅ All schedules deleted successfully")
				return nil
			}
			if deleted, ok := resp_body["deleted"]; ok {
				color.Green("✅ Deleted %v schedules", deleted)
			} else {
				color.Green("✅ All schedules deleted successfully")
			}
			return nil
		}

		// Case 2: Delete all schedules for a function with --all
		if deleteAllFlag {
			if len(args) == 0 {
				color.Red("✗ Function name required with --all flag")
				return fmt.Errorf("missing function name")
			}

			functionName := args[0]
			if !confirmAction(fmt.Sprintf("Are you sure you want to delete all schedules for function '%s'?", functionName)) {
				color.Yellow("✗ Operation cancelled")
				return nil
			}

			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/schedules?function=%s", serverAddr, functionName),
				nil,
			)
			if err != nil {
				color.Red("✗ Failed to delete schedules")
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
				color.Yellow("℧ No schedules found for function '%s'", functionName)
				return nil
			}

			if resp.StatusCode != http.StatusOK {
				var errResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
					if errMsg, ok := errResp["error"]; ok {
						color.Red("✗ Failed to delete schedules")
						return fmt.Errorf("%s", errMsg)
					}
				}
				color.Red("✗ Failed to delete schedules")
				return fmt.Errorf("server error: %s", resp.Status)
			}

			var resp_body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&resp_body); err != nil {
				color.Green("✅ Schedules deleted successfully for function '%s'", functionName)
				return nil
			}
			if deleted, ok := resp_body["deleted"]; ok {
				color.Green("✅ Deleted %v schedules for function '%s'", deleted, functionName)
			} else {
				color.Green("✅ Schedules deleted successfully for function '%s'", functionName)
			}
			return nil
		}

		// Case 3: Delete specific schedule by ID (current behavior)
		if len(args) == 0 {
			color.Red("✗ Schedule ID required (or use --all with function name or --removeall)")
			return fmt.Errorf("missing schedule ID or flags")
		}

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

// confirmAction prompts user for confirmation of a destructive action
func confirmAction(prompt string) bool {
	fmt.Print(color.YellowString(prompt + " (yes/no): "))
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}

func init() {
	// Flags
	scheduleCreateCmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	scheduleCreateCmd.Flags().StringVar(&data, "data", "", "JSON payload")

	scheduleListCmd.Flags().StringVarP(&functionFilter, "function", "f", "", "Filter by function name")

	scheduleDeleteCmd.Flags().BoolVar(&deleteAllFlag, "all", false, "Delete all schedules for a function")
	scheduleDeleteCmd.Flags().BoolVar(&deleteRemoveAllFlag, "removeall", false, "Delete all schedules globally")

	// Attach subcommands
	scheduleCmd.AddCommand(scheduleCreateCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleDeleteCmd)

	rootCmd.AddCommand(scheduleCmd)
}
