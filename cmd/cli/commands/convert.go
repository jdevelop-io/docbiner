package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	convertFormat    string
	convertOutput    string
	convertPageSize  string
	convertLandscape bool
	convertMargin    string
	convertHeader    string
	convertFooter    string
	convertWaitFor   string
	convertDelay     int
)

var convertCmd = &cobra.Command{
	Use:   "convert [source]",
	Short: "Convert HTML/URL to PDF or image",
	Long: `Convert an HTML file, URL, or stdin content to PDF, PNG, JPEG, or WebP.

Examples:
  docbiner convert https://example.com -o output.pdf
  docbiner convert input.html -f png -o screenshot.png
  echo '<h1>Hello</h1>' | docbiner convert - -o hello.pdf
  docbiner convert page.html --page-size Letter --landscape -o report.pdf`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertFormat, "format", "f", "pdf", "Output format: pdf, png, jpeg, webp")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Output file path (default: stdout)")
	convertCmd.Flags().StringVar(&convertPageSize, "page-size", "A4", "Page size: A4, Letter, Legal")
	convertCmd.Flags().BoolVar(&convertLandscape, "landscape", false, "Landscape orientation")
	convertCmd.Flags().StringVar(&convertMargin, "margin", "", "Margins (e.g., \"20mm\")")
	convertCmd.Flags().StringVar(&convertHeader, "header", "", "Header HTML")
	convertCmd.Flags().StringVar(&convertFooter, "footer", "", "Footer HTML")
	convertCmd.Flags().StringVar(&convertWaitFor, "wait-for", "", "CSS selector to wait for")
	convertCmd.Flags().IntVar(&convertDelay, "delay", 0, "Delay in milliseconds before conversion")

	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	source := args[0]

	// Build request payload
	payload := map[string]interface{}{
		"format":    convertFormat,
		"page_size": convertPageSize,
	}

	if convertLandscape {
		payload["landscape"] = true
	}
	if convertMargin != "" {
		payload["margin"] = convertMargin
	}
	if convertHeader != "" {
		payload["header_html"] = convertHeader
	}
	if convertFooter != "" {
		payload["footer_html"] = convertFooter
	}
	if convertWaitFor != "" {
		payload["wait_for"] = convertWaitFor
	}
	if convertDelay > 0 {
		payload["delay"] = convertDelay
	}

	// Determine source type
	switch {
	case source == "-":
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		payload["html"] = string(data)

	case isURL(source):
		payload["url"] = source

	default:
		// Read from file
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", source, err)
		}
		payload["html"] = string(data)
	}

	// Make API call
	result, err := client.postBinary("/v1/convert", payload)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Write output
	if convertOutput != "" {
		if err := os.WriteFile(convertOutput, result, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s (%d bytes)\n", convertOutput, len(result))
	} else {
		if _, err := os.Stdout.Write(result); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}

	return nil
}

func isURL(s string) bool {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		_, err := url.ParseRequestURI(s)
		return err == nil
	}
	return false
}
