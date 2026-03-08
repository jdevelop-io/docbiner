package renderer

// PDFOptions configures PDF generation via chromedp.
type PDFOptions struct {
	PageSize         string  `json:"page_size"`         // A4, Letter, etc.
	Landscape        bool    `json:"landscape"`
	MarginTop        string  `json:"margin_top"`        // e.g. "20mm"
	MarginBottom     string  `json:"margin_bottom"`
	MarginLeft       string  `json:"margin_left"`
	MarginRight      string  `json:"margin_right"`
	HeaderHTML       string  `json:"header_html"`
	FooterHTML       string  `json:"footer_html"`
	CSS              string  `json:"css"`
	JS               string  `json:"js"`
	WaitFor          string  `json:"wait_for"`          // CSS selector to wait for
	DelayMs          int     `json:"delay_ms"`
	Scale            float64 `json:"scale"`
	PrintBG          bool    `json:"print_background"`
	WatermarkText    string  `json:"watermark_text"`
	WatermarkOpacity float64 `json:"watermark_opacity"`
}

// ScreenshotOptions configures screenshot generation via chromedp.
type ScreenshotOptions struct {
	Format   string `json:"format"`    // png, jpeg, webp
	Quality  int    `json:"quality"`   // 0-100 for jpeg/webp
	FullPage bool   `json:"full_page"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	CSS      string `json:"css"`
	JS       string `json:"js"`
	WaitFor  string `json:"wait_for"`
	DelayMs  int    `json:"delay_ms"`
}
