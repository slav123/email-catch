package unit

import (
	"os"
	"testing"

	"github.com/slav123/email-catch/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadValidConfig(t *testing.T) {
	configData := `
server:
  ports: [2525, 2526]
  hostname: "localhost"
  tls:
    enabled: false
  rate_limit:
    enabled: false
    max_emails_per_minute: 100
    max_email_size_mb: 25

storage:
  s3_compatible:
    enabled: true
    endpoint: "localhost:9000"
    access_key: "test"
    secret_key: "test"
    bucket: "test-bucket"
    region: "us-east-1"
    use_ssl: false
  local:
    enabled: true
    directory: "./test-emails"

routes:
  - name: "test_route"
    condition:
      recipient_pattern: "test@.*"
    actions:
      - type: "store_local"
        enabled: true
        config:
          folder: "test"
    enabled: true

logging:
  level: "info"
  format: "json"
  file: ""
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, []int{2525, 2526}, cfg.Server.Ports)
	assert.Equal(t, "localhost", cfg.Server.Hostname)
	assert.False(t, cfg.Server.TLS.Enabled)
	assert.True(t, cfg.Storage.S3Compatible.Enabled)
	assert.Equal(t, "test-bucket", cfg.Storage.S3Compatible.Bucket)
	assert.True(t, cfg.Storage.Local.Enabled)
	assert.Equal(t, "./test-emails", cfg.Storage.Local.Directory)
	assert.Len(t, cfg.Routes, 1)
	assert.Equal(t, "test_route", cfg.Routes[0].Name)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := config.LoadConfig("nonexistent-config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadInvalidYAML(t *testing.T) {
	configData := `
server:
  ports: [2525
  hostname: "localhost"
invalid yaml structure
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestConfigValidationNoPorts(t *testing.T) {
	configData := `
server:
  ports: []
  hostname: "localhost"

storage:
  local:
    enabled: true
    directory: "./test"

routes:
  - name: "test"
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_local"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one server port must be specified")
}

func TestConfigValidationInvalidPort(t *testing.T) {
	configData := `
server:
  ports: [99999]
  hostname: "localhost"

storage:
  local:
    enabled: true
    directory: "./test"

routes:
  - name: "test"
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_local"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port number: 99999")
}

func TestConfigValidationNoStorageBackend(t *testing.T) {
	configData := `
server:
  ports: [2525]
  hostname: "localhost"

storage:
  s3_compatible:
    enabled: false
  local:
    enabled: false

routes:
  - name: "test"
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_local"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one storage backend must be enabled")
}

func TestConfigValidationS3Settings(t *testing.T) {
	configData := `
server:
  ports: [2525]
  hostname: "localhost"

storage:
  s3_compatible:
    enabled: true
    endpoint: ""
    bucket: ""
  local:
    enabled: false

routes:
  - name: "test"
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_s3"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 endpoint must be specified")
}

func TestConfigValidationRouteSettings(t *testing.T) {
	configData := `
server:
  ports: [2525]
  hostname: "localhost"

storage:
  local:
    enabled: true
    directory: "./test"

routes:
  - name: ""
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_local"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	_, err = config.LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have a name")
}

func TestGetEnabledRoutes(t *testing.T) {
	cfg := &config.Config{
		Routes: []config.RouteConfig{
			{Name: "route1", Enabled: true},
			{Name: "route2", Enabled: false},
			{Name: "route3", Enabled: true},
		},
	}

	enabled := cfg.GetEnabledRoutes()
	assert.Len(t, enabled, 2)
	assert.Equal(t, "route1", enabled[0].Name)
	assert.Equal(t, "route3", enabled[1].Name)
}

func TestConfigDefaultHostname(t *testing.T) {
	configData := `
server:
  ports: [2525]
  hostname: ""

storage:
  local:
    enabled: true
    directory: "./test"

routes:
  - name: "test"
    condition:
      recipient_pattern: ".*"
    actions:
      - type: "store_local"
        enabled: true
    enabled: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configData)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.Server.Hostname)
}