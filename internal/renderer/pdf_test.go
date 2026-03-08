package renderer

import (
	"strings"
	"testing"
)

func TestRenderHTMLToPDF(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello Docbiner</h1><p>This is a test page.</p></body>
</html>`

	pdf, err := r.HTMLToPDF(html, PDFOptions{
		PageSize: "A4",
		PrintBG:  true,
	})
	if err != nil {
		t.Fatalf("HTMLToPDF failed: %v", err)
	}

	if len(pdf) == 0 {
		t.Fatal("HTMLToPDF returned empty PDF")
	}

	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("HTMLToPDF output does not start with PDF magic bytes, got: %q", string(pdf[:min(len(pdf), 20)]))
	}
}

func TestRenderHTMLToPDFWithCSS(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>CSS Test</title></head>
<body><h1>Styled Page</h1></body>
</html>`

	pdf, err := r.HTMLToPDF(html, PDFOptions{
		PageSize: "A4",
		CSS:      "h1 { color: red; font-size: 48px; }",
		PrintBG:  true,
	})
	if err != nil {
		t.Fatalf("HTMLToPDF with CSS failed: %v", err)
	}

	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatal("HTMLToPDF with CSS output does not start with PDF magic bytes")
	}
}

func TestRenderHTMLToPDFWithWatermark(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>Watermark Test</title></head>
<body><h1>Watermarked Page</h1></body>
</html>`

	pdf, err := r.HTMLToPDF(html, PDFOptions{
		PageSize:         "A4",
		WatermarkText:    "DRAFT",
		WatermarkOpacity: 0.3,
	})
	if err != nil {
		t.Fatalf("HTMLToPDF with watermark failed: %v", err)
	}

	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatal("HTMLToPDF with watermark output does not start with PDF magic bytes")
	}
}

func TestRenderHTMLToPDFWithMargins(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>Margin Test</title></head>
<body><h1>Custom Margins</h1></body>
</html>`

	pdf, err := r.HTMLToPDF(html, PDFOptions{
		PageSize:     "Letter",
		Landscape:    true,
		MarginTop:    "25mm",
		MarginBottom: "25mm",
		MarginLeft:   "15mm",
		MarginRight:  "15mm",
	})
	if err != nil {
		t.Fatalf("HTMLToPDF with margins failed: %v", err)
	}

	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatal("HTMLToPDF with margins output does not start with PDF magic bytes")
	}
}

func TestRenderURLToPDF(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	pdf, err := r.URLToPDF("https://example.com", PDFOptions{
		PageSize: "A4",
	})
	if err != nil {
		t.Fatalf("URLToPDF failed: %v", err)
	}

	if len(pdf) == 0 {
		t.Fatal("URLToPDF returned empty PDF")
	}

	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("URLToPDF output does not start with PDF magic bytes, got: %q", string(pdf[:min(len(pdf), 20)]))
	}
}

func TestParseMM(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"25.4mm", 1.0},
		{"0mm", 0.0},
		{"50.8mm", 2.0},
		{"12.7mm", 0.5},
		{"", 0.0},
		{"invalid", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseMM(tt.input)
			if diff := got - tt.expected; diff > 0.001 || diff < -0.001 {
				t.Errorf("parseMM(%q) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}
