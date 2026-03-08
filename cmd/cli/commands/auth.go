package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Login, logout, and check authentication status for the Docbiner API.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your API key",
	Long:  "Store your Docbiner API key locally in ~/.docbiner/config.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your Docbiner API key: ")
		key, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		key = strings.TrimSpace(key)

		if key == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		cfg, err := loadConfig()
		if err != nil {
			cfg = &Config{}
		}

		cfg.APIKey = key
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.docbiner.com"
		}

		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("Authentication configured successfully.")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long:  "Remove the stored API key from ~/.docbiner/config.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := deleteConfig(); err != nil {
			return fmt.Errorf("failed to remove config: %w", err)
		}
		fmt.Println("Logged out successfully.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Long:  "Display the current authentication configuration and API key status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := getAPIKey()
		if key == "" {
			fmt.Println("Status: Not authenticated")
			fmt.Println("Run 'docbiner auth login' to authenticate.")
			return nil
		}

		masked := maskKey(key)
		fmt.Printf("Status: Authenticated\n")
		fmt.Printf("API Key: %s\n", masked)
		fmt.Printf("Base URL: %s\n", getBaseURL())

		// Determine source
		if apiKey != "" {
			fmt.Println("Source: --api-key flag")
		} else if loadConfigKey() != "" {
			fmt.Println("Source: ~/.docbiner/config.json")
		} else {
			fmt.Println("Source: DOCBINER_API_KEY environment variable")
		}

		return nil
	},
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
