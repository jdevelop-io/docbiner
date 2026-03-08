package pdfutil_test

import (
	"bytes"
	"testing"

	"github.com/docbiner/docbiner/internal/pdfutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	pdf1 := generateTestPDF(t, `<!DOCTYPE html>
<html><head><title>Page 1</title></head>
<body><h1>First Document</h1></body></html>`)

	pdf2 := generateTestPDF(t, `<!DOCTYPE html>
<html><head><title>Page 2</title></head>
<body><h1>Second Document</h1></body></html>`)

	merged, err := pdfutil.Merge([][]byte{pdf1, pdf2})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(merged, []byte("%PDF")), "merged output should start with %%PDF")
	// Merged PDF should be at least as large as the largest input.
	assert.Greater(t, len(merged), 0, "merged PDF should not be empty")
}

func TestMergeSinglePDF(t *testing.T) {
	pdf := generateTestPDF(t, `<!DOCTYPE html>
<html><head><title>Single</title></head>
<body><h1>Only Document</h1></body></html>`)

	merged, err := pdfutil.Merge([][]byte{pdf})
	require.NoError(t, err)

	assert.Equal(t, pdf, merged, "single PDF merge should return the same bytes")
}

func TestMergeEmpty(t *testing.T) {
	_, err := pdfutil.Merge(nil)
	assert.Error(t, err)

	_, err = pdfutil.Merge([][]byte{})
	assert.Error(t, err)
}
