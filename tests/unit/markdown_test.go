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
	// Alt text "Test Image" is used as the display name when available
	assert.Contains(t, markdown, "![Test Image](https://img.komunikacja.pro/2025/07/test_email/image1.jpg)")
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

	// Check that image is embedded as markdown image (alt text "Photo" is used as display name)
	assert.Contains(t, markdown, "![Photo](https://img.img.example.com/2025/07/test_email/photo.png)")
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

// TestReplaceImagesWithMarkdown tests the image replacement functionality
func TestReplaceImagesWithMarkdown(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		attachments    []email.Attachment
		wantContain    []string
		wantNotContain []string
	}{
		{
			name: "CID reference with matching filename",
			html: `<p>Image:</p><img src="cid:image1.jpg" alt="Test">`,
			attachments: []email.Attachment{
				{Filename: "image1.jpg", ContentType: "image/jpeg", Size: 1024},
			},
			wantContain:    []string{"![Test](https://img.komunikacja.pro/2025/07/test/image1.jpg)"},
			wantNotContain: []string{"<img"},
		},
		{
			name: "CID reference with URL-encoded filename matching alt text",
			html: `<img src="cid:ABC123@example" alt="Zrzut ekranu 2025-12-08 111218">`,
			attachments: []email.Attachment{
				{Filename: "Zrzut%20ekranu%202025-12-08%20111218[2].png", ContentType: "image/png", Size: 2048},
			},
			wantContain:    []string{"![Zrzut ekranu 2025-12-08 111218](https://img.komunikacja.pro/2025/07/test/Zrzut%20ekranu%202025-12-08%20111218[2].png)"},
			wantNotContain: []string{"<img"},
		},
		{
			name: "Multiple images with different CIDs",
			html: `<img src="cid:img1" alt="First"><p>text</p><img src="cid:img2" alt="Second">`,
			attachments: []email.Attachment{
				{Filename: "First.png", ContentType: "image/png", Size: 1024},
				{Filename: "Second.jpg", ContentType: "image/jpeg", Size: 2048},
			},
			wantContain: []string{
				"![First](https://img.komunikacja.pro/2025/07/test/First.png)",
				"![Second](https://img.komunikacja.pro/2025/07/test/Second.jpg)",
			},
			wantNotContain: []string{"<img"},
		},
		{
			name:           "HTTP URL image (external)",
			html:           `<img src="https://example.com/image.png" alt="External">`,
			attachments:    []email.Attachment{},
			wantContain:    []string{"![External](https://example.com/image.png)"},
			wantNotContain: []string{"<img"},
		},
		{
			name:           "HTTP URL image without alt text",
			html:           `<img src="https://example.com/image.png">`,
			attachments:    []email.Attachment{},
			wantContain:    []string{"![Image](https://example.com/image.png)"},
			wantNotContain: []string{"<img"},
		},
		{
			name: "Image without alt uses filename",
			html: `<img src="cid:photo.jpg">`,
			attachments: []email.Attachment{
				{Filename: "photo.jpg", ContentType: "image/jpeg", Size: 1024},
			},
			wantContain:    []string{"![photo.jpg](https://img.komunikacja.pro/2025/07/test/photo.jpg)"},
			wantNotContain: []string{"<img"},
		},
		{
			name: "Non-image attachment not converted to image markdown",
			html: `<img src="cid:document.pdf">`,
			attachments: []email.Attachment{
				{Filename: "document.pdf", ContentType: "application/pdf", Size: 1024},
			},
			// Non-image attachments are not converted to markdown images
			// The img tag gets stripped by HTML cleanup, but no ![...](...) is generated
			wantNotContain: []string{"![document.pdf]"},
		},
		{
			name: "Unmatched CID gets stripped",
			html: `<img src="cid:unknown123@example">`,
			attachments: []email.Attachment{
				{Filename: "other.jpg", ContentType: "image/jpeg", Size: 1024},
			},
			// Unmatched img tags get stripped by HTML cleanup (no markdown image generated)
			wantNotContain: []string{"![", "cid:unknown123"},
		},
		{
			name: "Complex img tag with multiple attributes",
			html: `<img width="1549" height="658" id="_x0000_i1025" alt="Screenshot 2025" class="proton-embedded" src="cid:ABC@DOMAIN">`,
			attachments: []email.Attachment{
				{Filename: "Screenshot%202025.png", ContentType: "image/png", Size: 209741},
			},
			wantContain:    []string{"![Screenshot 2025](https://img.komunikacja.pro/2025/07/test/Screenshot%202025.png)"},
			wantNotContain: []string{"<img", "width=", "height="},
		},
		{
			name: "Alt text with spaces matches URL-encoded filename",
			html: `<img src="cid:XYZ" alt="My Photo 2025">`,
			attachments: []email.Attachment{
				{Filename: "My%20Photo%202025.jpg", ContentType: "image/jpeg", Size: 5000},
			},
			wantContain: []string{"![My Photo 2025](https://img.komunikacja.pro/2025/07/test/My%20Photo%202025.jpg)"},
		},
		{
			name: "Empty alt attribute uses filename",
			html: `<img src="cid:test.png" alt="">`,
			attachments: []email.Attachment{
				{Filename: "test.png", ContentType: "image/png", Size: 1024},
			},
			wantContain: []string{"![test.png](https://img.komunikacja.pro/2025/07/test/test.png)"},
		},
		{
			name: "Content-ID header matching with renamed attachment",
			html: `<img src="cid:image001.png@01DC68FA.6F6ED740" alt="image001.png">`,
			attachments: []email.Attachment{
				{Filename: "attachment_1.png", ContentType: "image/png", Size: 90827, ContentID: "image001.png@01DC68FA.6F6ED740"},
			},
			wantContain:    []string{"![image001.png](https://img.komunikacja.pro/2025/07/test/attachment_1.png)"},
			wantNotContain: []string{"<img"},
		},
		{
			name: "Multiple inline images with Content-ID",
			html: `<img src="cid:image001.png@ABC" alt="First"><p>text</p><img src="cid:image002.png@DEF" alt="Second">`,
			attachments: []email.Attachment{
				{Filename: "attachment_1.png", ContentType: "image/png", Size: 1024, ContentID: "image001.png@ABC"},
				{Filename: "attachment_2.png", ContentType: "image/png", Size: 2048, ContentID: "image002.png@DEF"},
			},
			wantContain: []string{
				"![First](https://img.komunikacja.pro/2025/07/test/attachment_1.png)",
				"![Second](https://img.komunikacja.pro/2025/07/test/attachment_2.png)",
			},
			wantNotContain: []string{"<img"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := email.NewMarkdownConverter("https://img.example.com", "komunikacja-pro/2025/07/test")

			testEmail := &email.Email{
				From:        "test@example.com",
				To:          []string{"recipient@example.com"},
				Subject:     "Test",
				Date:        time.Date(2025, 7, 18, 12, 0, 0, 0, time.UTC),
				HTMLBody:    tt.html,
				Attachments: tt.attachments,
			}

			result := converter.ConvertToMarkdown(testEmail)

			for _, want := range tt.wantContain {
				assert.Contains(t, result, want, "Expected result to contain: %s", want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "Expected result NOT to contain: %s", notWant)
			}
		})
	}
}

// TestFindAttachmentBySrc tests the attachment matching logic
func TestFindAttachmentBySrc(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		attachments  []email.Attachment
		expectMatch  bool
		expectedFile string
	}{
		{
			name: "Direct filename match in CID",
			html: `<img src="cid:report.png" alt="Report">`,
			attachments: []email.Attachment{
				{Filename: "report.png", ContentType: "image/png", Size: 1024},
			},
			expectMatch:  true,
			expectedFile: "report.png",
		},
		{
			name: "Alt text matches decoded filename",
			html: `<img src="cid:RANDOM123" alt="Zrzut ekranu 2025-12-08 111218">`,
			attachments: []email.Attachment{
				{Filename: "Zrzut%20ekranu%202025-12-08%20111218[2].png", ContentType: "image/png", Size: 2048},
			},
			expectMatch:  true,
			expectedFile: "Zrzut%20ekranu%202025-12-08%20111218[2].png",
		},
		{
			name: "Alt text with normalized spaces matches",
			html: `<img src="cid:XYZ" alt="Test File Name">`,
			attachments: []email.Attachment{
				{Filename: "Test%20File%20Name.jpg", ContentType: "image/jpeg", Size: 1024},
			},
			expectMatch:  true,
			expectedFile: "Test%20File%20Name.jpg",
		},
		{
			name: "No match - img tag stripped by HTML cleanup",
			html: `<img src="cid:UNKNOWN" alt="Unknown Image">`,
			attachments: []email.Attachment{
				{Filename: "different.png", ContentType: "image/png", Size: 1024},
			},
			expectMatch: false,
		},
		{
			name: "Partial filename match in CID",
			html: `<img src="cid:screenshot.png@domain" alt="">`,
			attachments: []email.Attachment{
				{Filename: "screenshot.png", ContentType: "image/png", Size: 1024},
			},
			expectMatch:  true,
			expectedFile: "screenshot.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := email.NewMarkdownConverter("https://img.example.com", "komunikacja-pro/2025/07/test")

			testEmail := &email.Email{
				From:        "test@example.com",
				To:          []string{"recipient@example.com"},
				Subject:     "Test",
				Date:        time.Date(2025, 7, 18, 12, 0, 0, 0, time.UTC),
				HTMLBody:    tt.html,
				Attachments: tt.attachments,
			}

			result := converter.ConvertToMarkdown(testEmail)

			if tt.expectMatch {
				// Should contain markdown image with the expected filename
				assert.Contains(t, result, tt.expectedFile, "Expected filename in result")
				assert.Contains(t, result, "![", "Should contain markdown image syntax")
			} else {
				// Unmatched img tags get stripped by HTML cleanup - no markdown image generated
				assert.NotContains(t, result, "![", "Should not contain markdown image when no match")
			}
		})
	}
}
