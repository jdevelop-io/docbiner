package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GeneratedKey holds the result of an API key generation.
type GeneratedKey struct {
	Raw    string // Full key shown to user once: db_live_xxxxx
	Hash   string // SHA-256 hash stored in DB
	Prefix string // First 12 chars for identification: db_live_xxxx
}

// Generate creates a new API key with the given environment prefix (live or test).
// The raw key is composed of 32 random bytes encoded as hex, prefixed with db_{env}_.
func Generate(env string) (*GeneratedKey, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("apikey: generate random bytes: %w", err)
	}

	raw := fmt.Sprintf("db_%s_%s", env, hex.EncodeToString(b))
	hash := Hash(raw)
	prefix := raw[:12]

	return &GeneratedKey{
		Raw:    raw,
		Hash:   hash,
		Prefix: prefix,
	}, nil
}

// Hash returns the SHA-256 hex digest of the given raw API key.
func Hash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
