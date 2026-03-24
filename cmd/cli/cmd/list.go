package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type Function struct {
	ID      int    `json:"ID"`
	Name    string `json:"Name"`
	Version string `json:"Version"`
	Status  string `json:"Status"`
	Runtime string `json:"Runtime"`
}

type ListResponse struct {
	Functions []Function `json:"functions"`
}

var (
	activeBadge = color.New(color.FgGreen).Sprint("[active]")
	idleBadge   = color.New(color.FgHiBlack).Sprint("[idle]")

	headerColor = color.New(color.FgCyan, color.Bold)
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all functions",
	Args:  cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {

		url := fmt.Sprintf("%s/functions", serverAddr)

		resp, err := http.Get(url)
		if err != nil {
			color.Red("Failed to list functions")
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			color.Red("Failed to list functions")
			return fmt.Errorf("status: %s", resp.Status)
		}

		var result ListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			color.Red("Failed to parse response")
			return err
		}

		if len(result.Functions) == 0 {
			color.Yellow("No functions found")
			return nil
		}

		activeCount := 0
		idleCount := 0

		for _, fn := range result.Functions {
			if fn.Status == "active" {
				activeCount++
			} else {
				idleCount++
			}
		}

		headerColor.Printf("Functions • %d Found\n\n", len(result.Functions))

		for _, fn := range result.Functions {
			if fn.Status == "active" {
				fmt.Printf(" %-10s %s   %s\n",

					fn.Name,
					fn.Version,
					activeBadge,
				)
			} else {
				fmt.Printf(" %-10s %s   %s\n",

					fn.Name,
					fn.Version,
					idleBadge,
				)
			}
		}

		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
