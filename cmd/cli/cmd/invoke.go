/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
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
	// Args: cobra.ExactArgs(1),
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
		defer resp.Body.Close()

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
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// invokeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// invokeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	invokeCmd.Flags().StringVar(&functionName, "name", "", "Name of the function to invoke")
	invokeCmd.Flags().StringVar(&data, "data", "", "Data to pass to the function as input")

	invokeCmd.MarkFlagRequired("name")
	invokeCmd.MarkFlagRequired("data")
}
