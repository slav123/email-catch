package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/slav123/email-catch/internal/config"
	"github.com/slav123/email-catch/internal/storage"
	"github.com/slav123/email-catch/internal/webhook"
	"github.com/slav123/email-catch/pkg/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookPayloadGeneration(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			S3Compatible: config.S3Config{
				Enabled: false,
			},
			Local: config.LocalConfig{
				Enabled:   true,
				Directory: "/tmp/test-emails",
			},
		},
		Routes: []config.RouteConfig{
			{
				Name:    "faktury_hib",
				Enabled: true,
				Condition: config.Condition{
					RecipientPattern: "faktury@hib\\.pl",
				},
				Actions: []config.Action{
					{
						Type:    "store_s3",
						Enabled: true,
						Config: map[string]string{
							"folder": "faktury-hib",
						},
					},
					{
						Type:    "webhook",
						Enabled: true,
						Config: map[string]string{
							"url":    "https://n8n.gex.pl/webhook/test",
							"method": "POST",
						},
					},
				},
			},
		},
	}

	// Create storage backend
	storageBackend, err := storage.NewStorageBackend(cfg)
	require.NoError(t, err)

	// Create webhook client
	webhookClient := webhook.NewClient()

	// Create processor
	processor := email.NewProcessor(cfg, storageBackend, webhookClient)

	// Create test email similar to the forwarded email
	testEmail := &email.Email{
		From:      "Slawomir Jasinski <slav123@gmail.com>",
		To:        []string{"faktury@hib.pl"},
		Subject:   "Fwd: Test forwarded email",
		Date:      time.Date(2025, 7, 18, 13, 23, 2, 0, time.UTC),
		MessageID: "<CCDA6365-80CD-4CCD-AE8F-3C45BB2BB914@gmail.com>",
		Body:      "This is a forwarded email with content...",
		HTMLBody:  "<html><body>Forwarded email content</body></html>",
		Attachments: []email.Attachment{
			{
				Filename:    "forwarded_email.eml",
				ContentType: "message/rfc822",
				Content:     []byte("Original email content"),
				Size:        100,
			},
		},
	}

	// Test folder detection
	folder := processor.GetRouteFolder(testEmail)
	assert.Equal(t, "faktury-hib", folder)

	// Test filename generation
	filename := processor.GenerateFilename(testEmail)
	assert.Contains(t, filename, "20250718_132302")
	assert.Contains(t, filename, "CCDA6365-80CD-4CCD-AE8F-3C45BB2BB914")
	assert.True(t, strings.HasSuffix(filename, ".eml"))
}

func TestMarkdownGenerationForForwardedEmail(t *testing.T) {
	// Create test email with forwarded content
	testEmail := &email.Email{
		From:      "Slawomir Jasinski <slav123@gmail.com>",
		To:        []string{"faktury@hib.pl"},
		Subject:   "Fwd: Twoje pokwitowanie od Exafunction, Inc. nr 2322-5812",
		Date:      time.Date(2025, 7, 18, 13, 23, 2, 0, time.UTC),
		MessageID: "<CCDA6365-80CD-4CCD-AE8F-3C45BB2BB914@gmail.com>",
		Body:      "Begin forwarded message:\n\nFrom: Exafunction, Inc. <invoice@stripe.com>\nSubject: Your receipt\n\nThis is your receipt for $15.00",
		HTMLBody:  "",
		Attachments: []email.Attachment{
			{
				Filename:    "receipt.pdf",
				ContentType: "application/pdf",
				Content:     []byte("PDF content"),
				Size:        2048,
			},
		},
	}

	// Create markdown converter
	converter := email.NewMarkdownConverter("https://img.hib.pl", "faktury-hib/2025/07/test_email")
	markdown := converter.ConvertToMarkdown(testEmail)

	// Check markdown content
	assert.Contains(t, markdown, "# Email")
	assert.Contains(t, markdown, "**From:** Slawomir Jasinski <slav123@gmail.com>")
	assert.Contains(t, markdown, "**To:** faktury@hib.pl")
	assert.Contains(t, markdown, "**Subject:** Fwd: Twoje pokwitowanie od Exafunction, Inc. nr 2322-5812")
	assert.Contains(t, markdown, "Begin forwarded message:")
	assert.Contains(t, markdown, "## Attachments")
	assert.Contains(t, markdown, "[receipt.pdf](https://img.hib.pl/2025/07/test_email/receipt.pdf)")
	assert.Contains(t, markdown, "2.0 KB")
}