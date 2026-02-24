/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"faas-engine-go/internal/buildcontext"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy a function in the runtime manager",
	Long: `Example usage:
lambda deploy --file ./my-function --function-name my-function
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if filePath == "" {
			return fmt.Errorf("file path is required")
		}

		if functionName == "" {
			return fmt.Errorf("function name is required")
		}

		abspath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		//create a tar stream of the function directory
		tarstream, err := buildcontext.CreateTarStream(abspath)
		if err != nil {
			return fmt.Errorf("failed to create tar stream: %w", err)
		}

		//send the tarstream to the server
		Response, err := buildcontext.SendTarStream(tarstream, "http://localhost:8080/functions", functionName)
		if err != nil {
			slog.Info("failed to send tar stream",
				"error", err,
			)
			return fmt.Errorf("failed to send tar stream: %w", err)
		}

		slog.Info("response from server:",
			"Message", Response,
		)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&filePath, "file", "", "Path to the function code directory")
	deployCmd.Flags().StringVar(&functionName, "function-name", "", "Name of the function to deploy")

	deployCmd.MarkFlagRequired("file")
	deployCmd.MarkFlagRequired("function-name")
}
