package pdfutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// Merge combines multiple PDFs into a single PDF document.
// If only 1 PDF is provided, it is returned as-is.
// If 0 PDFs are provided, an error is returned.
func Merge(pdfs [][]byte) ([]byte, error) {
	switch len(pdfs) {
	case 0:
		return nil, fmt.Errorf("pdfutil: no PDFs to merge")
	case 1:
		return pdfs[0], nil
	}

	readers := make([]io.ReadSeeker, len(pdfs))
	for i, data := range pdfs {
		readers[i] = bytes.NewReader(data)
	}

	var buf bytes.Buffer

	if err := api.MergeRaw(readers, &buf, false, nil); err != nil {
		return nil, fmt.Errorf("pdfutil: merge failed: %w", err)
	}

	return buf.Bytes(), nil
}
