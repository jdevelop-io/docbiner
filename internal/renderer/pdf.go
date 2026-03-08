package renderer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// pageSizes maps common page size names to width and height in inches.
var pageSizes = map[string][2]float64{
	"A4":      {8.27, 11.69},
	"A3":      {11.69, 16.54},
	"A5":      {5.83, 8.27},
	"Letter":  {8.5, 11.0},
	"Legal":   {8.5, 14.0},
	"Tabloid": {11.0, 17.0},
}

// parseMM converts a margin string like "20mm" to inches.
// Returns 0.0 if the string is empty or invalid.
func parseMM(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0.0
	}

	s = strings.TrimSuffix(s, "mm")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}

	return v / 25.4
}

// HTMLToPDF renders the given HTML string to PDF bytes.
func (r *Renderer) HTMLToPDF(html string, opts PDFOptions) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	var buf []byte

	actions := []chromedp.Action{
		chromedp.Navigate("about:blank"),
		setDocumentContent(html),
	}

	actions = append(actions, buildInjectionActions(opts)...)
	actions = append(actions, printToPDFAction(opts, &buf))

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("renderer: HTMLToPDF failed: %w", err)
	}

	return buf, nil
}

// URLToPDF navigates to the given URL and renders the page to PDF bytes.
func (r *Renderer) URLToPDF(url string, opts PDFOptions) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	var buf []byte

	actions := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	}

	if opts.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitFor))
	}

	actions = append(actions, buildInjectionActions(opts)...)
	actions = append(actions, printToPDFAction(opts, &buf))

	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, fmt.Errorf("renderer: URLToPDF failed: %w", err)
	}

	return buf, nil
}

// setDocumentContent sets the page HTML using CDP's page.SetDocumentContent.
func setDocumentContent(html string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		frameTree, err := page.GetFrameTree().Do(ctx)
		if err != nil {
			return fmt.Errorf("renderer: failed to get frame tree: %w", err)
		}
		return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
	})
}

// buildInjectionActions returns chromedp actions for CSS, JS, watermark, and delay injection.
func buildInjectionActions(opts PDFOptions) []chromedp.Action {
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

	if opts.WatermarkText != "" {
		opacity := opts.WatermarkOpacity
		if opacity <= 0 {
			opacity = 0.15
		}
		js := fmt.Sprintf(`(() => {
			const wm = document.createElement('div');
			wm.textContent = %s;
			wm.style.cssText = 'position:fixed;top:50%%;left:50%%;transform:translate(-50%%,-50%%) rotate(-45deg);font-size:80px;color:rgba(0,0,0,%f);pointer-events:none;z-index:99999;white-space:nowrap;';
			document.body.appendChild(wm);
		})()`, quoteJS(opts.WatermarkText), opacity)
		actions = append(actions, chromedp.Evaluate(js, nil))
	}

	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	return actions
}

// printToPDFAction returns a chromedp action that calls page.PrintToPDF and writes the result to buf.
func printToPDFAction(opts PDFOptions, buf *[]byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Resolve page size.
		paperWidth, paperHeight := 8.27, 11.69 // A4 default
		if size, ok := pageSizes[opts.PageSize]; ok {
			paperWidth, paperHeight = size[0], size[1]
		}

		scale := opts.Scale
		if scale <= 0 {
			scale = 1.0
		}

		printParams := page.PrintToPDF().
			WithPaperWidth(paperWidth).
			WithPaperHeight(paperHeight).
			WithLandscape(opts.Landscape).
			WithScale(scale).
			WithMarginTop(parseMM(opts.MarginTop)).
			WithMarginBottom(parseMM(opts.MarginBottom)).
			WithMarginLeft(parseMM(opts.MarginLeft)).
			WithMarginRight(parseMM(opts.MarginRight)).
			WithPrintBackground(opts.PrintBG)

		if opts.HeaderHTML != "" || opts.FooterHTML != "" {
			printParams = printParams.WithDisplayHeaderFooter(true)
			if opts.HeaderHTML != "" {
				printParams = printParams.WithHeaderTemplate(opts.HeaderHTML)
			}
			if opts.FooterHTML != "" {
				printParams = printParams.WithFooterTemplate(opts.FooterHTML)
			}
		}

		data, _, err := printParams.Do(ctx)
		if err != nil {
			return fmt.Errorf("renderer: PrintToPDF failed: %w", err)
		}

		*buf = data
		return nil
	})
}

// quoteJS returns a JavaScript string literal safe for embedding in JS code.
func quoteJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return "'" + s + "'"
}
