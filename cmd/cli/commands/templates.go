package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage templates",
	Long:  "List, create, preview, and delete document templates.",
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.get("/v1/templates")
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result) == 0 {
			fmt.Println("No templates found.")
			return nil
		}

		for _, tmpl := range result {
			id := tmpl["id"]
			name := tmpl["name"]
			engine := tmpl["engine"]
			fmt.Printf("  %s  %s  (engine: %s)\n", id, name, engine)
		}

		return nil
	},
}

var templatesGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get template details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		data, err := client.get("/v1/templates/" + args[0])
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		formatted, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(formatted))

		return nil
	},
}

var (
	templateName   string
	templateEngine string
	templateFile   string
)

var templatesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new template",
	Long: `Create a new template from an HTML file.

Example:
  docbiner templates create --name "Invoice" --engine handlebars --file template.html`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		content, err := os.ReadFile(templateFile)
		if err != nil {
			return fmt.Errorf("failed to read template file: %w", err)
		}

		payload := map[string]interface{}{
			"name":    templateName,
			"engine":  templateEngine,
			"content": string(content),
		}

		data, err := client.post("/v1/templates", payload)
		if err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Template created: %s (id: %s)\n", result["name"], result["id"])
		return nil
	},
}

var templatePreviewData string

var templatesPreviewCmd = &cobra.Command{
	Use:   "preview [id]",
	Short: "Preview a template with sample data",
	Long: `Render a template with provided data and output the result.

Example:
  docbiner templates preview abc123 --data '{"title":"Test","items":[1,2,3]}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		var dataMap map[string]interface{}
		if templatePreviewData != "" {
			if err := json.Unmarshal([]byte(templatePreviewData), &dataMap); err != nil {
				return fmt.Errorf("invalid JSON data: %w", err)
			}
		}

		payload := map[string]interface{}{
			"data": dataMap,
		}

		result, err := client.postBinary("/v1/templates/"+args[0]+"/preview", payload)
		if err != nil {
			return fmt.Errorf("failed to preview template: %w", err)
		}

		if convertOutput != "" {
			if err := os.WriteFile(convertOutput, result, 0o644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Preview written to %s (%d bytes)\n", convertOutput, len(result))
		} else {
			os.Stdout.Write(result)
		}

		return nil
	},
}

var templatesDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		if err := client.delete("/v1/templates/" + args[0]); err != nil {
			return fmt.Errorf("failed to delete template: %w", err)
		}

		fmt.Printf("Template %s deleted.\n", args[0])
		return nil
	},
}

func init() {
	templatesCreateCmd.Flags().StringVar(&templateName, "name", "", "Template name (required)")
	templatesCreateCmd.Flags().StringVar(&templateEngine, "engine", "handlebars", "Template engine: handlebars, go")
	templatesCreateCmd.Flags().StringVar(&templateFile, "file", "", "Path to HTML template file (required)")
	templatesCreateCmd.MarkFlagRequired("name")
	templatesCreateCmd.MarkFlagRequired("file")

	templatesPreviewCmd.Flags().StringVar(&templatePreviewData, "data", "", "JSON data for template rendering")
	templatesPreviewCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Output file path")

	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesGetCmd)
	templatesCmd.AddCommand(templatesCreateCmd)
	templatesCmd.AddCommand(templatesPreviewCmd)
	templatesCmd.AddCommand(templatesDeleteCmd)

	rootCmd.AddCommand(templatesCmd)
}
