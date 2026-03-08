package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show current month usage and quota",
	Long: `Display your current plan, conversion usage, and remaining quota.

Example output:
  Plan: Pro
  Conversions: 450 / 15,000 (3.0%)
  Test conversions: 23
  Remaining: 14,550`,
	RunE: runUsage,
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runUsage(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	data, err := client.get("/v1/usage")
	if err != nil {
		return fmt.Errorf("failed to fetch usage: %w", err)
	}

	var result struct {
		Plan            string `json:"plan"`
		Conversions     int    `json:"conversions"`
		Quota           int    `json:"quota"`
		TestConversions int    `json:"test_conversions"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse usage data: %w", err)
	}

	remaining := result.Quota - result.Conversions
	if remaining < 0 {
		remaining = 0
	}

	var pct float64
	if result.Quota > 0 {
		pct = float64(result.Conversions) / float64(result.Quota) * 100
	}

	fmt.Printf("Plan: %s\n", result.Plan)
	fmt.Printf("Conversions: %s / %s (%.1f%%)\n",
		formatNumber(result.Conversions),
		formatNumber(result.Quota),
		pct,
	)
	fmt.Printf("Test conversions: %d\n", result.TestConversions)
	fmt.Printf("Remaining: %s\n", formatNumber(remaining))

	return nil
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		thousands := n / 1000
		remainder := n % 1000
		if remainder == 0 {
			return fmt.Sprintf("%d,000", thousands)
		}
		return fmt.Sprintf("%d,%03d", thousands, remainder)
	}
	return fmt.Sprintf("%d", n)
}
