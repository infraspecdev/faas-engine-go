package cmd

import (
	"faas-engine-go/internal/buildcontext"
	"fmt"
	"log/slog"
	"os"
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

	RunE: func(cmd *cobra.Command, args []string) error {
		abspath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		// create a tar stream of the function directory
		tarstream, err := buildcontext.CreateTarStream(abspath)
		if err != nil {
			return fmt.Errorf("failed to create tar stream: %w", err)
		}

		// send the tarstream to the server
		response, err := buildcontext.SendTarStream(
			tarstream,
			"http://localhost:8080/functions",
			functionName,
		)

		if err != nil {
			slog.Error("failed to send tar stream", "error", err)
			return err
		}

		slog.Info("response from server", "message", response)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&filePath, "file", "", "Path to the function code directory")
	deployCmd.Flags().StringVar(&functionName, "function-name", "", "Name of the function to deploy")

	if err := deployCmd.MarkFlagRequired("file"); err != nil {
		slog.Error("failed to mark flag as required", "flag", "file", "error", err)
		os.Exit(1)
	}

	if err := deployCmd.MarkFlagRequired("function-name"); err != nil {
		slog.Error("failed to mark flag as required", "flag", "function-name", "error", err)
		os.Exit(1)
	}
}
