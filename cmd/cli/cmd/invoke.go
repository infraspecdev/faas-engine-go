package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var invokeCmd = &cobra.Command{
	Use:   "invoke <function-name>",
	Short: "invoke a function in the runtime",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		functionName := strings.TrimSpace(args[0])
		if functionName == "" {
			return fmt.Errorf("function name is required")
		}

		url := fmt.Sprintf("%s/functions/%s/invoke", serverAddr, functionName)

		req, err := http.NewRequest("POST", url, strings.NewReader(data))
		if err != nil {
			color.Red("Invoke failed")
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 15 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			color.Red("Invoke failed")
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("failed to close response body: %v", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			color.Red("Invoke failed")
			return err
		}

		if resp.StatusCode == http.StatusNotFound ||
			strings.Contains(strings.ToLower(string(body)), "function not found") {

			color.Yellow("Function not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			color.Red("Invoke failed")
			return fmt.Errorf("%s", string(body))
		}

		fmt.Println(string(body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(invokeCmd)

	invokeCmd.Flags().StringVar(&data, "data", "", "JSON data to pass to the function")
}
