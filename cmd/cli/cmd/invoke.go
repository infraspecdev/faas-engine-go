/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// invokeCmd represents the invoke command
var invokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "invoke a function in the runtime",
	Long: `Invoke command allows you to execute a deployed function in the runtime manager.
Example usage:
lambda invoke --name my-function
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if functionName == "" {
			return fmt.Errorf("function name is required")
		}

		url := fmt.Sprintf("%s/functions/%s/invoke", serverAddr, functionName)

		req, err := http.NewRequest("POST", url, strings.NewReader(data))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 15 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				slog.Error("failed to close response body", "error", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("invoke failed: %s", string(body))
		}

		fmt.Println(string(body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(invokeCmd)

	invokeCmd.Flags().StringVar(&functionName, "name", "", "Name of the function to invoke")
	invokeCmd.Flags().StringVar(&data, "data", "", "Data to pass to the function as input")

	if err := invokeCmd.MarkFlagRequired("name"); err != nil {
		slog.Error("failed to mark flag as required", "flag", "name", "error", err)
		os.Exit(1)
	}

	if err := invokeCmd.MarkFlagRequired("data"); err != nil {
		slog.Error("failed to mark flag as required", "flag", "data", "error", err)
		os.Exit(1)
	}
}
