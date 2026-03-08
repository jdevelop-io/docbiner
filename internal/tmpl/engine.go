package tmpl

import (
	"fmt"

	"github.com/aymerick/raymond"
	"github.com/osteele/liquid"
)

// Render renders the given template string using the specified engine and data.
// Supported engines are "handlebars" and "liquid".
func Render(engine string, template string, data map[string]interface{}) (string, error) {
	switch engine {
	case "handlebars":
		return renderHandlebars(template, data)
	case "liquid":
		return renderLiquid(template, data)
	default:
		return "", fmt.Errorf("tmpl: unknown engine %q", engine)
	}
}

// renderHandlebars renders a Handlebars template with the given data.
func renderHandlebars(tpl string, data map[string]interface{}) (string, error) {
	result, err := raymond.Render(tpl, data)
	if err != nil {
		return "", fmt.Errorf("tmpl: handlebars render failed: %w", err)
	}
	return result, nil
}

// renderLiquid renders a Liquid template with the given data.
func renderLiquid(tpl string, data map[string]interface{}) (string, error) {
	eng := liquid.NewEngine()
	bindings := liquid.Bindings(data)

	result, err := eng.ParseAndRenderString(tpl, bindings)
	if err != nil {
		return "", fmt.Errorf("tmpl: liquid render failed: %w", err)
	}
	return result, nil
}
