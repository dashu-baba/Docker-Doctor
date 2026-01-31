package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/example/docker-doctor/internal/collector"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the Docker host and generate a JSON report",
	Long: `Scan the Docker host to collect metadata about the host, Docker daemon,
containers, images, volumes, and disk usage. Outputs the report in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		return runScan(output)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scanCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	scanCmd.Flags().StringP("output", "o", "", "Output file for the JSON report (default stdout)")
}

func runScan(output string) error {
	ctx := context.Background()

	report, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect data: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if output == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("Report written to %s\n", output)
	}

	return nil
}