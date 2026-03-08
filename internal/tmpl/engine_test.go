package tmpl

import (
	"testing"
)

func TestRender_Handlebars_Variables(t *testing.T) {
	tpl := "<h1>{{title}}</h1><p>{{description}}</p>"
	data := map[string]interface{}{
		"title":       "My Invoice",
		"description": "This is a test invoice",
	}

	result, err := Render("handlebars", tpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<h1>My Invoice</h1><p>This is a test invoice</p>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_Handlebars_Conditionals(t *testing.T) {
	tpl := "{{#if show}}<p>visible</p>{{/if}}"
	data := map[string]interface{}{
		"show": true,
	}

	result, err := Render("handlebars", tpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<p>visible</p>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_Liquid_Variables(t *testing.T) {
	tpl := "<h1>{{ title }}</h1><p>{{ description }}</p>"
	data := map[string]interface{}{
		"title":       "My Invoice",
		"description": "This is a test invoice",
	}

	result, err := Render("liquid", tpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<h1>My Invoice</h1><p>This is a test invoice</p>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_Liquid_Loop(t *testing.T) {
	tpl := "{% for item in items %}<li>{{ item }}</li>{% endfor %}"
	data := map[string]interface{}{
		"items": []interface{}{"A", "B", "C"},
	}

	result, err := Render("liquid", tpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<li>A</li><li>B</li><li>C</li>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_UnknownEngine(t *testing.T) {
	_, err := Render("jinja", "<p>test</p>", nil)
	if err == nil {
		t.Fatal("expected error for unknown engine, got nil")
	}

	expected := `tmpl: unknown engine "jinja"`
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestRender_Handlebars_InvalidTemplate(t *testing.T) {
	// Unclosed block helper should cause a parse error.
	tpl := "{{#if show}}<p>unclosed"
	data := map[string]interface{}{
		"show": true,
	}

	_, err := Render("handlebars", tpl, data)
	if err == nil {
		t.Fatal("expected error for invalid handlebars template, got nil")
	}
}

func TestRender_Liquid_InvalidTemplate(t *testing.T) {
	// Unclosed tag should cause a parse error.
	tpl := "{% for item in items %}<li>{{ item }}</li>"
	data := map[string]interface{}{
		"items": []interface{}{"A"},
	}

	_, err := Render("liquid", tpl, data)
	if err == nil {
		t.Fatal("expected error for invalid liquid template, got nil")
	}
}

func TestRender_EmptyData(t *testing.T) {
	tpl := "<h1>{{title}}</h1>"
	data := map[string]interface{}{}

	result, err := Render("handlebars", tpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<h1></h1>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_NilData(t *testing.T) {
	tpl := "<p>Hello</p>"

	result, err := Render("handlebars", tpl, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<p>Hello</p>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
