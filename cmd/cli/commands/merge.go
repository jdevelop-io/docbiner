package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var mergeOutput string

var mergeCmd = &cobra.Command{
	Use:   "merge [sources...]",
	Short: "Merge multiple sources into a single PDF",
	Long: `Convert and merge multiple HTML files or URLs into a single PDF document.

Each source is converted to PDF individually, then all pages are merged
into a single output file.

Examples:
  docbiner merge file1.html file2.html -o merged.pdf
  docbiner merge file1.html https://example.com file2.html -o merged.pdf`,
	Args: cobra.MinimumNArgs(2),
	RunE: runMerge,
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeOutput, "output", "o", "", "Output file path (required)")
	mergeCmd.MarkFlagRequired("output")

	rootCmd.AddCommand(mergeCmd)
}

func runMerge(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	sources := make([]map[string]interface{}, 0, len(args))

	for _, source := range args {
		entry := map[string]interface{}{}

		switch {
		case source == "-":
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			entry["html"] = string(data)

		case isURL(source):
			entry["url"] = source

		default:
			data, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", source, err)
			}
			entry["html"] = string(data)
		}

		sources = append(sources, entry)
	}

	payload := map[string]interface{}{
		"sources": sources,
		"format":  "pdf",
	}

	result, err := client.postBinary("/v1/merge", payload)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	if err := os.WriteFile(mergeOutput, result, 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Merged %d sources into %s (%d bytes)\n", len(args), mergeOutput, len(result))
	return nil
}
