package unit

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/slav123/email-catch/internal/config"
	"github.com/slav123/email-catch/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageBackendCompression(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			S3Compatible: config.S3Config{
				Enabled: false, // Disable S3 for unit test
			},
			Local: config.LocalConfig{
				Enabled:   true,
				Directory: "/tmp/test-emails",
			},
		},
	}

	backend, err := storage.NewStorageBackend(cfg)
	require.NoError(t, err)

	// Test data - use longer content for better compression
	testData := []byte(`This is a test email content that should be compressed when stored as .eml or .json file. 
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. 
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. 
Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. 
Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
This text repeats to ensure compression is effective. This text repeats to ensure compression is effective.
This text repeats to ensure compression is effective. This text repeats to ensure compression is effective.`)

	// Test compression function directly
	storageBackend, ok := backend.(*storage.StorageBackend)
	require.True(t, ok, "Backend should be of type StorageBackend")
	
	// Test shouldCompress function
	assert.True(t, storageBackend.ShouldCompress("test.eml", "message/rfc822"))
	assert.True(t, storageBackend.ShouldCompress("test.json", "application/json"))
	assert.False(t, storageBackend.ShouldCompress("test.pdf", "application/pdf"))
	assert.False(t, storageBackend.ShouldCompress("test.jpg", "image/jpeg"))

	// Test compression
	compressed, err := storageBackend.CompressData(testData)
	require.NoError(t, err)
	assert.True(t, len(compressed) < len(testData), "Compressed data should be smaller")

	// Test decompression to verify integrity
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	var decompressed bytes.Buffer
	_, err = decompressed.ReadFrom(reader)
	require.NoError(t, err)

	assert.Equal(t, testData, decompressed.Bytes(), "Decompressed data should match original")
}