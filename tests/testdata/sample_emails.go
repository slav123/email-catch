package testdata

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type EmailTemplate struct {
	From        string
	To          []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []AttachmentTemplate
}

type AttachmentTemplate struct {
	Filename    string
	ContentType string
	Content     []byte
}

func GenerateSimpleEmail() EmailTemplate {
	return EmailTemplate{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Simple Test Email",
		Body:    "This is a simple test email with plain text content.",
	}
}

func GenerateHTMLEmail() EmailTemplate {
	return EmailTemplate{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "HTML Test Email",
		Body:     "This is the plain text version of the email.",
		HTMLBody: "<html><body><h1>HTML Test Email</h1><p>This is the <strong>HTML</strong> version of the email.</p></body></html>",
	}
}

func GenerateEmailWithAttachments() EmailTemplate {
	return EmailTemplate{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Email with Attachments",
		Body:    "This email contains multiple attachments.",
		Attachments: []AttachmentTemplate{
			{
				Filename:    "document.txt",
				ContentType: "text/plain",
				Content:     []byte("This is a text document attachment."),
			},
			{
				Filename:    "data.csv",
				ContentType: "text/csv",
				Content:     []byte("Name,Age,City\nJohn,30,New York\nJane,25,London"),
			},
			{
				Filename:    "image.png",
				ContentType: "image/png",
				Content:     generateFakePNGData(),
			},
		},
	}
}

func GenerateLargeEmail() EmailTemplate {
	largeBody := strings.Repeat("This is a line of text in a large email. ", 1000)
	largeAttachment := bytes.Repeat([]byte("Large file content. "), 50000)

	return EmailTemplate{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Large Email Test",
		Body:    largeBody,
		Attachments: []AttachmentTemplate{
			{
				Filename:    "large_file.bin",
				ContentType: "application/octet-stream",
				Content:     largeAttachment,
			},
		},
	}
}

func GenerateMultiRecipientEmail() EmailTemplate {
	return EmailTemplate{
		From:    "sender@example.com",
		To:      []string{"recipient1@example.com", "recipient2@example.com", "recipient3@example.com"},
		Subject: "Multi-Recipient Email",
		Body:    "This email is sent to multiple recipients.",
	}
}

func GenerateEmailWithSpecialCharacters() EmailTemplate {
	return EmailTemplate{
		From:    "sender@exämple.com",
		To:      []string{"recipiënt@exámple.com"},
		Subject: "Tëst Ëmáil with Spëciål Chäractërs 🚀",
		Body:    "This email contains special characters: àáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ and emojis: 🎉🔥💯",
	}
}

func GenerateEmailWithLongSubject() EmailTemplate {
	longSubject := "This is a very long email subject that exceeds the typical length recommendations for email subjects and should test how the system handles very long subject lines that might cause issues with parsing or storage"

	return EmailTemplate{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: longSubject,
		Body:    "This email has a very long subject line.",
	}
}

func (t EmailTemplate) ToRawEmail() []byte {
	var email strings.Builder

	email.WriteString(fmt.Sprintf("From: %s\r\n", t.From))
	email.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(t.To, ", ")))
	email.WriteString(fmt.Sprintf("Subject: %s\r\n", t.Subject))
	email.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	email.WriteString(fmt.Sprintf("Message-Id: <%d@example.com>\r\n", time.Now().Unix()))
	email.WriteString("MIME-Version: 1.0\r\n")

	if len(t.Attachments) > 0 {
		boundary := fmt.Sprintf("boundary_%d", time.Now().Unix())
		email.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		email.WriteString("\r\n")

		email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if t.HTMLBody != "" {
			email.WriteString("Content-Type: multipart/alternative; boundary=\"alt_boundary\"\r\n")
			email.WriteString("\r\n")
			
			email.WriteString("--alt_boundary\r\n")
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.Body)
			email.WriteString("\r\n")

			email.WriteString("--alt_boundary\r\n")
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.HTMLBody)
			email.WriteString("\r\n")

			email.WriteString("--alt_boundary--\r\n")
		} else {
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.Body)
			email.WriteString("\r\n")
		}

		for _, attachment := range t.Attachments {
			email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			email.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
			email.WriteString("Content-Transfer-Encoding: base64\r\n")
			email.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
			email.WriteString("\r\n")
			
			encoded := base64.StdEncoding.EncodeToString(attachment.Content)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				email.WriteString(encoded[i:end] + "\r\n")
			}
		}

		email.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		if t.HTMLBody != "" {
			email.WriteString("Content-Type: multipart/alternative; boundary=\"alt_boundary\"\r\n")
			email.WriteString("\r\n")
			
			email.WriteString("--alt_boundary\r\n")
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.Body)
			email.WriteString("\r\n")

			email.WriteString("--alt_boundary\r\n")
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.HTMLBody)
			email.WriteString("\r\n")

			email.WriteString("--alt_boundary--\r\n")
		} else {
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(t.Body)
		}
	}

	return []byte(email.String())
}

func generateFakePNGData() []byte {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	
	fakeData := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 100)
	
	return append(pngHeader, fakeData...)
}

func GetAllTestEmails() []EmailTemplate {
	return []EmailTemplate{
		GenerateSimpleEmail(),
		GenerateHTMLEmail(),
		GenerateEmailWithAttachments(),
		GenerateLargeEmail(),
		GenerateMultiRecipientEmail(),
		GenerateEmailWithSpecialCharacters(),
		GenerateEmailWithLongSubject(),
	}
}