package renderer

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// screenshotFormat maps format strings to chromedp page.CaptureScreenshotFormat values.
var screenshotFormat = map[string]page.CaptureScreenshotFormat{
	"png":  page.CaptureScreenshotFormatPng,
	"jpeg": page.CaptureScreenshotFormatJpeg,
	"webp": page.CaptureScreenshotFormatWebp,
}

// HTMLToScreenshot renders the given HTML string to image bytes (PNG, JPEG, or WebP).
func (r *Renderer) HTMLToScreenshot(html string, opts ScreenshotOptions) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	opts = applyScreenshotDefaults(opts)

	var buf []byte

	actions := []chromedp.Action{
		setDeviceMetrics(opts.Width, opts.Height),
		chromedp.Navigate("about:blank"),
		setDocumentContent(html),
		chromedp.WaitReady("body"),
	}

	actions = append(actions, buildScreenshotInjectionActions(opts)...)
	actions = append(actions, captureScreenshotAction(opts, &buf))

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("renderer: HTMLToScreenshot failed: %w", err)
	}

	return buf, nil
}

// URLToScreenshot navigates to the given URL and captures a screenshot as image bytes.
func (r *Renderer) URLToScreenshot(url string, opts ScreenshotOptions) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	opts = applyScreenshotDefaults(opts)

	var buf []byte

	actions := []chromedp.Action{
		setDeviceMetrics(opts.Width, opts.Height),
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	}

	if opts.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitFor))
	}

	actions = append(actions, buildScreenshotInjectionActions(opts)...)
	actions = append(actions, captureScreenshotAction(opts, &buf))

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("renderer: URLToScreenshot failed: %w", err)
	}

	return buf, nil
}

// applyScreenshotDefaults fills in zero-valued options with sensible defaults.
func applyScreenshotDefaults(opts ScreenshotOptions) ScreenshotOptions {
	if opts.Width <= 0 {
		opts.Width = 1280
	}
	if opts.Height <= 0 {
		opts.Height = 720
	}
	if opts.Format == "" {
		opts.Format = "png"
	}
	if opts.Quality <= 0 && (opts.Format == "jpeg" || opts.Format == "webp") {
		opts.Quality = 90
	}
	return opts
}

// setDeviceMetrics returns a chromedp action that overrides the device screen dimensions.
func setDeviceMetrics(width, height int) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1.0, false).Do(ctx)
	})
}

// buildScreenshotInjectionActions returns chromedp actions for CSS, JS, and delay injection.
func buildScreenshotInjectionActions(opts ScreenshotOptions) []chromedp.Action {
	var actions []chromedp.Action

	if opts.CSS != "" {
		js := fmt.Sprintf(`(() => {
			const style = document.createElement('style');
			style.textContent = %s;
			document.head.appendChild(style);
		})()`, quoteJS(opts.CSS))
		actions = append(actions, chromedp.Evaluate(js, nil))
	}

	if opts.JS != "" {
		actions = append(actions, chromedp.Evaluate(opts.JS, nil))
	}

	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	return actions
}

// captureScreenshotAction returns a chromedp action that captures a screenshot into buf.
func captureScreenshotAction(opts ScreenshotOptions, buf *[]byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		format, ok := screenshotFormat[opts.Format]
		if !ok {
			format = page.CaptureScreenshotFormatPng
		}

		if opts.FullPage {
			quality := opts.Quality
			// chromedp.FullScreenshot takes quality as int64; for PNG quality is ignored by Chrome.
			data, err := page.CaptureScreenshot().
				WithFormat(format).
				WithQuality(int64(quality)).
				WithCaptureBeyondViewport(true).
				WithFromSurface(true).
				Do(ctx)
			if err != nil {
				return fmt.Errorf("renderer: FullScreenshot failed: %w", err)
			}
			*buf = data
			return nil
		}

		// Viewport-only screenshot.
		params := page.CaptureScreenshot().
			WithFormat(format).
			WithFromSurface(true)

		if format != page.CaptureScreenshotFormatPng {
			params = params.WithQuality(int64(opts.Quality))
		}

		data, err := params.Do(ctx)
		if err != nil {
			return fmt.Errorf("renderer: CaptureScreenshot failed: %w", err)
		}

		*buf = data
		return nil
	})
}
