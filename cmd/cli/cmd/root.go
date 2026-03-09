/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	serverAddr   string
	filePath     string
	functionName string
	data         string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lambda",
	Short: "CLI for interacting with the FaaS runtime manager",
	Long:  `Lambda CLI allows deploying, invoking, listing and managing serverless functions.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cmd.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env file, using default configuration")
	}
	targetUrl := os.Getenv("PROXY_URL")
	if targetUrl == "" {
		targetUrl = "http://localhost"
	}
	targetPort := os.Getenv("PROXY_PORT")
	if targetPort == "" {
		targetPort = "80"
	}
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVar(
		&serverAddr,
		"server",
		targetUrl+":"+targetPort,
		"Address of the runtime manager server",
	)
}
