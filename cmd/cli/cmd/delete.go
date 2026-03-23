package cmd

import (
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <function-name>",
	Short: "Delete a function",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		functionName := args[0]

		url := fmt.Sprintf("%s/functions/%s", serverAddr, functionName)

		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			color.Red("Delete failed")
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			color.Red("Delete failed")
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			color.Yellow("Function not found")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			color.Red("Delete failed")
			return fmt.Errorf("status: %s", resp.Status)
		}

		color.Green("Deleted function: %s", functionName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
