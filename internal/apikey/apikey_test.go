package apikey_test

import (
	"testing"

	"github.com/docbiner/docbiner/internal/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	key, err := apikey.Generate("live")
	require.NoError(t, err)
	assert.True(t, len(key.Raw) > 20)
	assert.Equal(t, "db_live_", key.Raw[:8])
	assert.NotEmpty(t, key.Hash)
	assert.Equal(t, "db_live_", key.Prefix[:8])
}

func TestGenerateTestKey(t *testing.T) {
	key, err := apikey.Generate("test")
	require.NoError(t, err)
	assert.Equal(t, "db_test_", key.Raw[:8])
}

func TestHashAndVerify(t *testing.T) {
	key, _ := apikey.Generate("live")
	hash := apikey.Hash(key.Raw)
	assert.Equal(t, key.Hash, hash)
}
