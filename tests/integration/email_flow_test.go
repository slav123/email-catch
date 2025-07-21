package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slav123/email-catch/internal/config"
	"github.com/slav123/email-catch/internal/smtp"
	"github.com/slav123/email-catch/internal/storage"
	"github.com/slav123/email-catch/internal/webhook"
	"github.com/slav123/email-catch/pkg/email"
	"github.com/slav123/email-catch/tests/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailFlowIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "email-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := createTestConfig(tempDir)
	
	storageBackend, err := storage.NewStorageBackend(cfg)
	require.NoError(t, err)

	webhookClient := webhook.NewClient()
	processor := email.NewProcessor(cfg, storageBackend, webhookClient)

	server := smtp.NewServer(cfg, processor)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	smtpClient := client.NewSMTPClient("localhost", 2525)

	message := client.EmailMessage{
		From:    "test@example.com",
		To:      []string{"capture@test.com"},
		Subject: "Integration Test Email",
		Body:    "This is a test email from integration test.",
	}

	err = smtpClient.SendEmail(message)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	files, err := filepath.Glob(filepath.Join(tempDir, "capture", "*.eml"))
	require.NoError(t, err)
	assert.Len(t, files, 1, "Expected exactly one .eml file to be created")

	emailData, err := os.ReadFile(files[0])
	require.NoError(t, err)

	parsedEmail, err := email.ParseEmail(emailData, "test@example.com", []string{"capture@test.com"})
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", parsedEmail.From)
	assert.Equal(t, []string{"capture@test.com"}, parsedEmail.To)
	assert.Equal(t, "Integration Test Email", parsedEmail.Subject)
	assert.Contains(t, parsedEmail.Body, "This is a test email from integration test.")
}

func TestEmailWithAttachmentsFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "email-attachment-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := createTestConfig(tempDir)
	
	storageBackend, err := storage.NewStorageBackend(cfg)
	require.NoError(t, err)

	webhookClient := webhook.NewClient()
	processor := email.NewProcessor(cfg, storageBackend, webhookClient)

	server := smtp.NewServer(cfg, processor)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	smtpClient := client.NewSMTPClient("localhost", 2525)

	message := client.EmailMessage{
		From:    "sender@example.com",
		To:      []string{"attachments@test.com"},
		Subject: "Email with Attachment",
		Body:    "This email has attachments.",
		Attachments: []client.Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     []byte("This is a test attachment."),
			},
			{
				Filename:    "image.jpg",
				ContentType: "image/jpeg",
				Content:     []byte("fake-jpeg-data"),
			},
		},
	}

	err = smtpClient.SendEmail(message)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	files, err := filepath.Glob(filepath.Join(tempDir, "attachments", "*.eml"))
	require.NoError(t, err)
	assert.Len(t, files, 1, "Expected exactly one .eml file to be created")

	emailData, err := os.ReadFile(files[0])
	require.NoError(t, err)

	parsedEmail, err := email.ParseEmail(emailData, "sender@example.com", []string{"attachments@test.com"})
	require.NoError(t, err)

	assert.True(t, parsedEmail.HasAttachments())
	assert.Len(t, parsedEmail.Attachments, 2)
	
	testAttachment := parsedEmail.GetAttachmentByName("test.txt")
	require.NotNil(t, testAttachment)
	assert.Equal(t, "This is a test attachment.", string(testAttachment.Content))

	imageAttachment := parsedEmail.GetAttachmentByName("image.jpg")
	require.NotNil(t, imageAttachment)
	assert.Equal(t, "fake-jpeg-data", string(imageAttachment.Content))
}

func TestMultiPortEmailFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "email-multiport-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := createTestConfig(tempDir)
	cfg.Server.Ports = []int{2525, 2526, 2527}

	storageBackend, err := storage.NewStorageBackend(cfg)
	require.NoError(t, err)

	webhookClient := webhook.NewClient()
	processor := email.NewProcessor(cfg, storageBackend, webhookClient)

	server := smtp.NewServer(cfg, processor)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	ports := []int{2525, 2526, 2527}
	for _, port := range ports {
		smtpClient := client.NewSMTPClient("localhost", port)

		message := client.EmailMessage{
			From:    "test@example.com",
			To:      []string{"capture@test.com"},
			Subject: "Multi-Port Test Email",
			Body:    "This email was sent to port " + string(rune(port)),
		}

		err = smtpClient.SendEmail(message)
		require.NoError(t, err, "Failed to send email to port %d", port)

		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	files, err := filepath.Glob(filepath.Join(tempDir, "capture", "*.eml"))
	require.NoError(t, err)
	assert.Len(t, files, len(ports), "Expected %d .eml files to be created", len(ports))
}

func createTestConfig(tempDir string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Ports:    []int{2525},
			Hostname: "localhost",
			TLS: config.TLSConfig{
				Enabled: false,
			},
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
		},
		Storage: config.StorageConfig{
			S3Compatible: config.S3Config{
				Enabled: false,
			},
			Local: config.LocalConfig{
				Enabled:   true,
				Directory: tempDir,
			},
		},
		Routes: []config.RouteConfig{
			{
				Name: "capture_route",
				Condition: config.Condition{
					RecipientPattern: "capture@.*",
				},
				Actions: []config.Action{
					{
						Type:    "store_local",
						Enabled: true,
						Config: map[string]string{
							"folder": "capture",
						},
					},
				},
				Enabled: true,
			},
			{
				Name: "attachment_route",
				Condition: config.Condition{
					RecipientPattern: "attachments@.*",
				},
				Actions: []config.Action{
					{
						Type:    "store_local",
						Enabled: true,
						Config: map[string]string{
							"folder": "attachments",
						},
					},
				},
				Enabled: true,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
	}
}