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

	"github.com/fatih/color"
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
		fmt.Print("[1/3] Packaging function code...")
		tarstream, err := buildcontext.CreateTarStream(abspath, runtimeName)
		if err != nil {
			color.Red(" Failed. \n\n%s\n", err.Error())
			return nil
		}

		if _, err := color.New(color.FgGreen).Println(" Done."); err != nil {
			return fmt.Errorf("failed to print success message: %w", err)
		}

		//send the tarstream to the server
		url := fmt.Sprintf("%s/functions", serverAddr)

		// Stream deploy logs from server
		err = buildcontext.SendTarStream(tarstream, url, functionName)
		if err != nil {
			slog.Error("deployment failed", "error", err)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&filePath, "file", "", "Path to the function code directory")
	deployCmd.Flags().StringVar(&functionName, "function-name", "", "Name of the function to deploy")

	deployCmd.Flags().StringVar(&runtimeName, "runtime", "", "Name of the runtime to use")

	if err := deployCmd.MarkFlagRequired("file"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
	if err := deployCmd.MarkFlagRequired("function-name"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
	if err := deployCmd.MarkFlagRequired("runtime"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
}
