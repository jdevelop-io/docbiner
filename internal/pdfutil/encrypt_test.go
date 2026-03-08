package pdfutil_test

import (
	"bytes"
	"testing"

	"github.com/docbiner/docbiner/internal/pdfutil"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestPDF(t *testing.T, html string) []byte {
	t.Helper()

	r, err := renderer.New()
	require.NoError(t, err)
	defer r.Close()

	pdf, err := r.HTMLToPDF(html, renderer.PDFOptions{PageSize: "A4"})
	require.NoError(t, err)
	require.True(t, len(pdf) > 0, "generated PDF should not be empty")

	return pdf
}

func TestEncrypt(t *testing.T) {
	pdf := generateTestPDF(t, `<!DOCTYPE html>
<html><head><title>Encrypt Test</title></head>
<body><h1>Hello Encryption</h1></body></html>`)

	encrypted, err := pdfutil.Encrypt(pdf, pdfutil.EncryptOptions{
		UserPassword: "secret123",
	})
	require.NoError(t, err)

	// Encrypted PDF must still be a valid PDF.
	assert.True(t, bytes.HasPrefix(encrypted, []byte("%PDF")), "encrypted output should start with %%PDF")

	// Encrypted PDF must differ from the original.
	assert.False(t, bytes.Equal(pdf, encrypted), "encrypted PDF should differ from original")
}

func TestEncryptWithRestrictions(t *testing.T) {
	pdf := generateTestPDF(t, `<!DOCTYPE html>
<html><head><title>Restrict Test</title></head>
<body><h1>Restricted PDF</h1></body></html>`)

	encrypted, err := pdfutil.Encrypt(pdf, pdfutil.EncryptOptions{
		UserPassword:  "user",
		OwnerPassword: "owner",
		Restrict:      []string{"print", "copy"},
	})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(encrypted, []byte("%PDF")), "encrypted output should start with %%PDF")
	assert.False(t, bytes.Equal(pdf, encrypted), "encrypted PDF should differ from original")
}

func TestEncryptEmptyData(t *testing.T) {
	_, err := pdfutil.Encrypt(nil, pdfutil.EncryptOptions{
		UserPassword: "test",
	})
	assert.Error(t, err)
}
