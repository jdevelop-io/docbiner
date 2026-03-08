package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	apiKey  string
	baseURL string
)

var rootCmd = &cobra.Command{
	Use:   "docbiner",
	Short: "Docbiner CLI — HTML to PDF/Images conversion",
	Long: `Official CLI tool for Docbiner API.
Convert HTML/URLs to PDF, PNG, JPEG, WebP.

Authenticate with:
  docbiner auth login

Then convert documents:
  docbiner convert https://example.com -o output.pdf
  docbiner convert input.html -f png -o screenshot.png
  echo '<h1>Hello</h1>' | docbiner convert - -o hello.pdf`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (or set DOCBINER_API_KEY env)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "API base URL (default https://api.docbiner.com)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func getAPIKey() string {
	if apiKey != "" {
		return apiKey
	}
	if key := loadConfigKey(); key != "" {
		return key
	}
	return os.Getenv("DOCBINER_API_KEY")
}

func getBaseURL() string {
	if baseURL != "" {
		return baseURL
	}
	if url := loadConfigBaseURL(); url != "" {
		return url
	}
	if url := os.Getenv("DOCBINER_BASE_URL"); url != "" {
		return url
	}
	return "https://api.docbiner.com"
}
