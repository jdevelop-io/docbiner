package renderer

import (
	"testing"
)

func TestRenderHTMLToScreenshot(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>Screenshot Test</title></head>
<body><h1>Hello Docbiner</h1><p>This is a screenshot test.</p></body>
</html>`

	img, err := r.HTMLToScreenshot(html, ScreenshotOptions{
		Format: "png",
		Width:  1280,
		Height: 720,
	})
	if err != nil {
		t.Fatalf("HTMLToScreenshot failed: %v", err)
	}

	if len(img) == 0 {
		t.Fatal("HTMLToScreenshot returned empty image")
	}

	// PNG magic bytes: 0x89 0x50 0x4E 0x47 (i.e. \x89PNG)
	if img[0] != 0x89 || img[1] != 0x50 {
		t.Fatalf("HTMLToScreenshot output does not start with PNG magic bytes, got: %x %x", img[0], img[1])
	}
}

func TestRenderHTMLToScreenshotJPEG(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	html := `<!DOCTYPE html>
<html>
<head><title>JPEG Test</title></head>
<body><h1>JPEG Screenshot</h1></body>
</html>`

	img, err := r.HTMLToScreenshot(html, ScreenshotOptions{
		Format:  "jpeg",
		Quality: 80,
		Width:   800,
		Height:  600,
	})
	if err != nil {
		t.Fatalf("HTMLToScreenshot (JPEG) failed: %v", err)
	}

	if len(img) == 0 {
		t.Fatal("HTMLToScreenshot (JPEG) returned empty image")
	}

	// JPEG magic bytes: 0xFF 0xD8
	if img[0] != 0xFF || img[1] != 0xD8 {
		t.Fatalf("HTMLToScreenshot JPEG output does not start with JPEG magic bytes, got: %x %x", img[0], img[1])
	}
}

func TestRenderHTMLToScreenshotFullPage(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	// Long HTML to test full page capture
	html := `<!DOCTYPE html>
<html>
<head><title>Full Page Test</title></head>
<body>
<div style="height: 3000px; background: linear-gradient(red, blue);">
<h1>Full Page Screenshot</h1>
</div>
</body>
</html>`

	img, err := r.HTMLToScreenshot(html, ScreenshotOptions{
		Format:   "png",
		FullPage: true,
		Width:    1280,
		Height:   720,
	})
	if err != nil {
		t.Fatalf("HTMLToScreenshot (FullPage) failed: %v", err)
	}

	if len(img) == 0 {
		t.Fatal("HTMLToScreenshot (FullPage) returned empty image")
	}

	if img[0] != 0x89 || img[1] != 0x50 {
		t.Fatalf("HTMLToScreenshot (FullPage) output does not start with PNG magic bytes, got: %x %x", img[0], img[1])
	}
}

func TestRenderURLToScreenshot(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	defer r.Close()

	img, err := r.URLToScreenshot("https://example.com", ScreenshotOptions{
		Format: "png",
		Width:  1280,
		Height: 720,
	})
	if err != nil {
		t.Fatalf("URLToScreenshot failed: %v", err)
	}

	if len(img) == 0 {
		t.Fatal("URLToScreenshot returned empty image")
	}

	// PNG magic bytes
	if img[0] != 0x89 || img[1] != 0x50 {
		t.Fatalf("URLToScreenshot output does not start with PNG magic bytes, got: %x %x", img[0], img[1])
	}
}
