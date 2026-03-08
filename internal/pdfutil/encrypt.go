package pdfutil

import (
	"bytes"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// EncryptOptions configures PDF encryption.
type EncryptOptions struct {
	UserPassword  string   // Password to open the PDF
	OwnerPassword string   // Password for full permissions (if empty, same as UserPassword)
	Restrict      []string // Permissions to restrict: "print", "copy", "modify"
}

// Encrypt encrypts the given PDF bytes using AES-256 and returns the encrypted PDF.
func Encrypt(pdfData []byte, opts EncryptOptions) ([]byte, error) {
	if len(pdfData) == 0 {
		return nil, fmt.Errorf("pdfutil: empty PDF data")
	}

	ownerPW := opts.OwnerPassword
	if ownerPW == "" {
		ownerPW = opts.UserPassword
	}

	conf := model.NewAESConfiguration(opts.UserPassword, ownerPW, 256)
	conf.Permissions = applyRestrictions(opts.Restrict)

	rs := bytes.NewReader(pdfData)
	var buf bytes.Buffer

	if err := api.Encrypt(rs, &buf, conf); err != nil {
		return nil, fmt.Errorf("pdfutil: encryption failed: %w", err)
	}

	return buf.Bytes(), nil
}

// applyRestrictions converts a list of restriction names to a pdfcpu PermissionFlags value.
// By default all permissions are granted; restrictions remove specific ones.
func applyRestrictions(restrict []string) model.PermissionFlags {
	perms := model.PermissionsAll

	for _, r := range restrict {
		switch r {
		case "print":
			perms &^= model.PermissionPrintRev2 | model.PermissionPrintRev3
		case "copy":
			perms &^= model.PermissionExtract
		case "modify":
			perms &^= model.PermissionModify
		}
	}

	return perms
}
