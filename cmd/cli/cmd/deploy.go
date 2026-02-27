/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"faas-engine-go/internal/buildcontext"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy a function in the runtime manager",
	Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		response, err := buildcontext.SendTarStream(tarstream, "http://localhost:8080/functions", functionName)
		if err != nil {
			slog.Info("failed to send tar stream", "error", err)
			return err
		}

		slog.Info("response from server:", "Message", response)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&filePath, "file", "", "Path to the function code directory")
	deployCmd.Flags().StringVar(&functionName, "function-name", "", "Name of the function to deploy")

	if err := deployCmd.MarkFlagRequired("file"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
	if err := deployCmd.MarkFlagRequired("function-name"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
}
