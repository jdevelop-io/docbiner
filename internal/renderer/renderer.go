package renderer

import (
	"context"
	"os"

	"github.com/chromedp/chromedp"
)

// Renderer manages a shared chromedp exec allocator for generating PDFs and screenshots.
type Renderer struct {
	allocCtx context.Context
	cancel   context.CancelFunc
}

// New creates a Renderer with a chromedp exec allocator configured for headless rendering.
func New() (*Renderer, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	if p := os.Getenv("CHROMIUM_PATH"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	return &Renderer{
		allocCtx: allocCtx,
		cancel:   cancel,
	}, nil
}

// Close releases the chromedp allocator context and all associated resources.
func (r *Renderer) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}
