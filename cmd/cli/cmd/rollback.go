package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <function-name> [target-version]",
	Short: "Rollback a function to a previous version",
	Long: `Rollback a function to a specific version or to the previous version.

Examples:
  lambda rollback myfunction v1       # Rollback to version v1
`,
	Args: cobra.RangeArgs(1, 2),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := strings.TrimSpace(args[0])
		targetVersion := ""
		if len(args) > 1 {
			targetVersion = strings.TrimSpace(args[1])
		}

		url := fmt.Sprintf("%s/functions/%s/rollback", serverAddr, functionName)

		body := map[string]string{}
		if targetVersion != "" {
			body["target_version"] = targetVersion
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request: %w", err)
		}

		resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonBody)))
		if err != nil {
			color.Red("Rollback failed")
			return err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			color.Red("Rollback failed")
			return err
		}

		if resp.StatusCode == http.StatusNotFound {
			color.Yellow("Function or version not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		color.Green("✓ Rollback successful!")
		fmt.Printf("  Function: %v\n", result["function_name"])
		fmt.Printf("  From: %v\n", result["previous_version"])
		fmt.Printf("  To: %v\n", result["current_version"])
		fmt.Printf("  Time: %v\n", result["rolled_back_at"])

		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history <function-name>",
	Short: "Show rollback history for a function",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := strings.TrimSpace(args[0])

		url := fmt.Sprintf("%s/functions/%s/history", serverAddr, functionName)

		resp, err := http.Get(url)
		if err != nil {
			color.Red("Failed to fetch history")
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			color.Yellow("Function not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
		}

		var history []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(history) == 0 {
			color.Yellow("No rollback history found")
			return nil
		}

		color.Cyan("Rollback History for %s:\n", functionName)
		for i, entry := range history {
			fmt.Printf("%d. %v → %v (at: %v)\n",
				i+1,
				entry["from"],
				entry["to"],
				entry["at"],
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(historyCmd)
}
