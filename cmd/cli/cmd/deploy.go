/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"faas-engine-go/internal/buildcontext"
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
	Run: func(cmd *cobra.Command, args []string) {
		if filePath == "" {
			panic("file path is required")
		}

		if functionName == "" {
			panic("function name is required")
		}

		abspath, err := filepath.Abs(filePath)
		if err != nil {
			panic("failed to get absolute path: " + err.Error())
		}
		//create a tar stream of the function directory
		tarstream, err := buildcontext.CreateTarStream(abspath)
		if err != nil {
			panic("failed to create tar stream: " + err.Error())
		}

		//send the tarstream to the server
		Response, err := buildcontext.SendTarStream(tarstream, "http://localhost:8080/functions", functionName)
		if err != nil {
			slog.Info("failed to send tar stream", "error", err)
			return
		}

		slog.Info("response from server:", "Message", Response)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&filePath, "file", "", "Path to the function code directory")
	deployCmd.Flags().StringVar(&functionName, "function-name", "", "Name of the function to deploy")

	deployCmd.MarkFlagRequired("file")
	deployCmd.MarkFlagRequired("function-name")
}
