package unit

import (
	"testing"
	"time"

	"github.com/slav123/email-catch/pkg/email"
	"github.com/stretchr/testify/assert"
)

func TestMarkdownConverter(t *testing.T) {
	converter := email.NewMarkdownConverter("https://img.example.com", "komunikacja-pro/2025/07/test_email")

	// Create test email
	testEmail := &email.Email{
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		Subject:   "Test Email with HTML",
		Date:      time.Date(2025, 7, 18, 12, 46, 30, 0, time.UTC),
		MessageID: "<test@example.com>",
		Body:      "Plain text body",
		HTMLBody:  `<html><body><h1>Hello World</h1><p>This is a <strong>test</strong> email with <a href="https://example.com">link</a>.</p><img src="cid:image1.jpg" alt="Test Image"><br><div>Some content in div</div></body></html>`,
		Attachments: []email.Attachment{
			{
				Filename:    "image1.jpg",
				ContentType: "image/jpeg",
				Content:     []byte("fake image data"),
				Size:        1024,
			},
			{
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Content:     []byte("fake pdf data"),
				Size:        2048,
			},
		},
	}

	markdown := converter.ConvertToMarkdown(testEmail)

	// Check that markdown contains expected elements
	assert.Contains(t, markdown, "# Email")
	assert.Contains(t, markdown, "**From:** sender@example.com")
	assert.Contains(t, markdown, "**To:** recipient@example.com")
	assert.Contains(t, markdown, "**Subject:** Test Email with HTML")
	assert.Contains(t, markdown, "**Date:** 2025-07-18 12:46:30")
	assert.Contains(t, markdown, "**Message-ID:** <test@example.com>")
	assert.Contains(t, markdown, "# Hello World")
	assert.Contains(t, markdown, "**test**")
	assert.Contains(t, markdown, "[link](https://example.com)")
	assert.Contains(t, markdown, "![image1.jpg](https://img.komunikacja.pro/2025/07/test_email/image1.jpg)")
	assert.Contains(t, markdown, "## Attachments")
	assert.Contains(t, markdown, "[document.pdf](https://img.komunikacja.pro/2025/07/test_email/document.pdf)")
	assert.Contains(t, markdown, "application/pdf")
	assert.Contains(t, markdown, "2.0 KB")
}

func TestMarkdownConverterPlainText(t *testing.T) {
	converter := email.NewMarkdownConverter("https://img.example.com", "test-folder/2025/07/test_email")

	// Create test email with plain text only
	testEmail := &email.Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Plain Text Email",
		Date:    time.Date(2025, 7, 18, 12, 46, 30, 0, time.UTC),
		Body:    "This is plain text content.\nWith multiple lines.",
	}

	markdown := converter.ConvertToMarkdown(testEmail)

	// Check that markdown contains expected elements
	assert.Contains(t, markdown, "# Email")
	assert.Contains(t, markdown, "**From:** sender@example.com")
	assert.Contains(t, markdown, "This is plain text content.")
	assert.Contains(t, markdown, "With multiple lines.")
	assert.NotContains(t, markdown, "## Attachments")
}

func TestMarkdownConverterAttachmentURL(t *testing.T) {
	converter := email.NewMarkdownConverter("https://img.example.com", "faktury-hib/2025/07/test_email")

	// Create test email with attachment
	testEmail := &email.Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Email with Attachment",
		Date:    time.Date(2025, 7, 18, 12, 46, 30, 0, time.UTC),
		Body:    "Email with attachment",
		Attachments: []email.Attachment{
			{
				Filename:    "invoice.pdf",
				ContentType: "application/pdf",
				Content:     []byte("fake pdf data"),
				Size:        5120,
			},
		},
	}

	markdown := converter.ConvertToMarkdown(testEmail)

	// Check that URL uses correct subdomain for faktury-hib
	assert.Contains(t, markdown, "https://img.hib.pl/2025/07/test_email/invoice.pdf")
	assert.Contains(t, markdown, "5.0 KB")
}

func TestMarkdownConverterImageEmbedding(t *testing.T) {
	converter := email.NewMarkdownConverter("https://img.example.com", "test-folder/2025/07/test_email")

	// Create test email with image in HTML
	testEmail := &email.Email{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "Email with Image",
		Date:     time.Date(2025, 7, 18, 12, 46, 30, 0, time.UTC),
		HTMLBody: `<html><body><p>Check out this image:</p><img src="cid:photo.png" alt="Photo"><p>End of email</p></body></html>`,
		Attachments: []email.Attachment{
			{
				Filename:    "photo.png",
				ContentType: "image/png",
				Content:     []byte("fake png data"),
				Size:        3072,
			},
		},
	}

	markdown := converter.ConvertToMarkdown(testEmail)

	// Check that image is embedded as markdown image
	assert.Contains(t, markdown, "![photo.png](https://img.img.example.com/2025/07/test_email/photo.png)")
	assert.Contains(t, markdown, "Check out this image:")
	assert.Contains(t, markdown, "End of email")
	// Image should not appear in attachments section since it's embedded
	assert.NotContains(t, markdown, "## Attachments")
}

func TestMarkdownConverterFileSizeFormatting(t *testing.T) {
	converter := email.NewMarkdownConverter("https://img.example.com", "test-folder/2025/07/test_email")

	// Create test email with different sized attachments
	testEmail := &email.Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Email with Various Attachments",
		Date:    time.Date(2025, 7, 18, 12, 46, 30, 0, time.UTC),
		Body:    "Email with various sized attachments",
		Attachments: []email.Attachment{
			{
				Filename:    "small.txt",
				ContentType: "text/plain",
				Content:     []byte("small content"),
				Size:        500,
			},
			{
				Filename:    "medium.pdf",
				ContentType: "application/pdf",
				Content:     []byte("medium content"),
				Size:        1536000, // 1.5 MB
			},
			{
				Filename:    "large.zip",
				ContentType: "application/zip",
				Content:     []byte("large content"),
				Size:        1073741824, // 1 GB
			},
		},
	}

	markdown := converter.ConvertToMarkdown(testEmail)

	// Check file size formatting
	assert.Contains(t, markdown, "500 B")
	assert.Contains(t, markdown, "1.5 MB")
	assert.Contains(t, markdown, "1.0 GB")
}