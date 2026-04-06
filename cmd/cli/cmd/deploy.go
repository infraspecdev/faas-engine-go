package cmd

import (
	"bufio"
	"encoding/json"
	"faas-engine-go/internal/buildcontext"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Deploy flags
var forceRedeploy bool

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy <path>",
	Short: "deploy a function in the runtime manager",
	Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		// ✅ take path from args instead of flag
		filePath := args[0]

		abspath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		//create a tar stream of the function directory
		fmt.Print("[1/5] Packaging function code...")
		tarstream, err := buildcontext.CreateTarStream(abspath, runtimeName)
		if err != nil {
			color.Red(" Failed. \n\n%s\n", err.Error())
			return fmt.Errorf("failed to package function code: %w", err)
		}

		if _, err := color.New(color.FgGreen).Println(" Done."); err != nil {
			return fmt.Errorf("failed to print success message: %w", err)
		}

		// Check if function already exists and ask for confirmation if not using --force
		fmt.Print("[2/5] Checking if function exists...")
		functionExists, err := checkFunctionExists(functionName)
		if err != nil {
			color.Red(" Failed. \n\n%s\n", err.Error())
			return fmt.Errorf("failed to check if function exists: %w", err)
		}

		if functionExists && !forceRedeploy {
			if _, err := color.New(color.FgYellow).Println(" Found."); err != nil {
				return fmt.Errorf("failed to print status message: %w", err)
			}

			// Ask for confirmation
			if !confirmReplacement(functionName) {
				color.Yellow("✗ Deployment cancelled")
				return nil
			}
		} else if functionExists {
			if _, err := color.New(color.FgYellow).Println(" Found (--force enabled)."); err != nil {
				return fmt.Errorf("failed to print status message: %w", err)
			}
		} else {
			if _, err := color.New(color.FgGreen).Println(" Not found."); err != nil {
				return fmt.Errorf("failed to print status message: %w", err)
			}
		}

		//send the tarstream to the server
		url := fmt.Sprintf("%s/functions", serverAddr)

		// Stream deploy logs from server
		fmt.Print("[3/5] Deploying function...")
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

	deployCmd.Flags().StringVar(&functionName, "name", "", "Name of the function to deploy")
	deployCmd.Flags().StringVar(&runtimeName, "runtime", "", "Name of the runtime to use")
	deployCmd.Flags().BoolVar(&forceRedeploy, "force", false, "Force redeploy without confirmation if function already exists")

	if err := deployCmd.MarkFlagRequired("name"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
	if err := deployCmd.MarkFlagRequired("runtime"); err != nil {
		log.Fatalf("failed to mark flag as required: %v", err)
	}
}

// checkFunctionExists queries the runtime-manager to see if a function with this name exists
func checkFunctionExists(functionName string) (bool, error) {
	url := fmt.Sprintf("%s/functions", serverAddr)

	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("unable to reach runtime manager at %s: %w", serverAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to list functions: server returned %s", resp.Status)
	}

	var response struct {
		Functions []struct {
			Name string `json:"name"`
		} `json:"functions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("failed to parse functions list: %w", err)
	}

	for _, fn := range response.Functions {
		if fn.Name == functionName {
			return true, nil
		}
	}

	return false, nil
}

// confirmReplacement prompts the user to confirm redeploying an existing function
func confirmReplacement(functionName string) bool {
	fmt.Print(color.YellowString(fmt.Sprintf("Function '%s' already exists. Replace it? (yes/no): ", functionName)))
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}
