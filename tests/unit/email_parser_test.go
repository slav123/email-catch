package unit

import (
	"testing"
	"time"

	"github.com/slav123/email-catch/pkg/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleEmail(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Email
Date: Mon, 02 Jan 2006 15:04:05 -0700
Message-Id: <123@example.com>

This is a test email body.`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Equal(t, "sender@example.com", parsedEmail.From)
	assert.Equal(t, []string{"recipient@example.com"}, parsedEmail.To)
	assert.Equal(t, "Test Email", parsedEmail.Subject)
	assert.Equal(t, "<123@example.com>", parsedEmail.MessageID)
	assert.Contains(t, parsedEmail.Body, "This is a test email body.")
	assert.False(t, parsedEmail.HasAttachments())
}

func TestParseEmailWithAttachment(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Email with Attachment
Date: Mon, 02 Jan 2006 15:04:05 -0700
Message-Id: <456@example.com>
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=UTF-8

This is the email body.

--boundary123
Content-Type: text/plain; charset=UTF-8
Content-Disposition: attachment; filename="test.txt"
Content-Transfer-Encoding: base64

VGhpcyBpcyBhIHRlc3QgYXR0YWNobWVudC4=

--boundary123--`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Equal(t, "sender@example.com", parsedEmail.From)
	assert.Equal(t, "Email with Attachment", parsedEmail.Subject)
	assert.True(t, parsedEmail.HasAttachments())
	assert.Len(t, parsedEmail.Attachments, 1)

	attachment := parsedEmail.Attachments[0]
	assert.Equal(t, "test.txt", attachment.Filename)
	assert.Equal(t, "text/plain; charset=UTF-8", attachment.ContentType)
	assert.Equal(t, "This is a test attachment.", string(attachment.Content))
}

func TestParseHTMLEmail(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: HTML Email
Date: Mon, 02 Jan 2006 15:04:05 -0700
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="boundary456"

--boundary456
Content-Type: text/plain; charset=UTF-8

This is the plain text version.

--boundary456
Content-Type: text/html; charset=UTF-8

<html><body><h1>This is the HTML version</h1></body></html>

--boundary456--`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Equal(t, "HTML Email", parsedEmail.Subject)
	assert.Contains(t, parsedEmail.Body, "This is the plain text version.")
	assert.Contains(t, parsedEmail.HTMLBody, "<h1>This is the HTML version</h1>")
}

func TestEmailSummary(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Summary
Date: Mon, 02 Jan 2006 15:04:05 -0700

Test body`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	summary := parsedEmail.Summary()
	assert.Contains(t, summary, "sender@example.com")
	assert.Contains(t, summary, "recipient@example.com")
	assert.Contains(t, summary, "Test Summary")
	assert.Contains(t, summary, "Attachments: 0")
}

func TestEmailToEML(t *testing.T) {
	rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Test EML

Test body`)

	parsedEmail, err := email.ParseEmail(rawEmail, "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	eml := parsedEmail.ToEML()
	assert.Equal(t, rawEmail, eml)
}

func TestEmailGetTotalSize(t *testing.T) {
	rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Test Size

Test body`)

	parsedEmail, err := email.ParseEmail(rawEmail, "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	size := parsedEmail.GetTotalSize()
	assert.Equal(t, int64(len(rawEmail)), size)
}

func TestEmailWithMultipleAttachments(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Multiple Attachments
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary789"

--boundary789
Content-Type: text/plain

Email body here.

--boundary789
Content-Type: text/plain
Content-Disposition: attachment; filename="file1.txt"
Content-Transfer-Encoding: base64

RmlsZSAxIGNvbnRlbnQ=

--boundary789
Content-Type: application/pdf
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

UERGIGNvbnRlbnQ=

--boundary789--`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Len(t, parsedEmail.Attachments, 2)
	assert.Equal(t, "file1.txt", parsedEmail.Attachments[0].Filename)
	assert.Equal(t, "document.pdf", parsedEmail.Attachments[1].Filename)
	assert.Equal(t, "File 1 content", string(parsedEmail.Attachments[0].Content))
	assert.Equal(t, "PDF content", string(parsedEmail.Attachments[1].Content))

	file1 := parsedEmail.GetAttachmentByName("file1.txt")
	assert.NotNil(t, file1)
	assert.Equal(t, "file1.txt", file1.Filename)

	nonExistent := parsedEmail.GetAttachmentByName("nonexistent.txt")
	assert.Nil(t, nonExistent)
}

func TestEmailDateParsing(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Date Test
Date: Tue, 15 Jan 2019 21:24:17 +0000

Test body`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	expectedDate := time.Date(2019, 1, 15, 21, 24, 17, 0, time.UTC)
	assert.True(t, parsedEmail.Date.Equal(expectedDate))
}

func TestEmailInvalidDate(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Invalid Date Test
Date: Invalid Date String

Test body`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.True(t, time.Since(parsedEmail.Date) < time.Minute)
}

func TestEmailHeaders(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Header Test
X-Custom-Header: custom-value
X-Priority: 1

Test body`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Equal(t, []string{"custom-value"}, parsedEmail.Headers["X-Custom-Header"])
	assert.Equal(t, []string{"1"}, parsedEmail.Headers["X-Priority"])
	assert.Equal(t, []string{"sender@example.com"}, parsedEmail.Headers["From"])
}

func TestMIMEHeaderDecoding(t *testing.T) {
	rawEmail := `From: =?UTF-8?Q?S=C5=82awomir_Jasi=C5=84ski?= <slawomir@jasinski.us>
To: recipient@example.com
Subject: =?UTF-8?B?VGVzdCBzdWJqZWN0IHdpdGggUG9saXNoIGNoYXJhY3RlcnMgxJTEmcSZxIXFhMW7?=
Date: Mon, 02 Jan 2006 15:04:05 -0700
Message-Id: <test@example.com>

Test body with Polish characters`

	parsedEmail, err := email.ParseEmail([]byte(rawEmail), "sender@example.com", []string{"recipient@example.com"})
	require.NoError(t, err)

	assert.Equal(t, "Sławomir Jasiński <slawomir@jasinski.us>", parsedEmail.From)
	assert.Equal(t, "Test subject with Polish characters ĔęęąńŻ", parsedEmail.Subject)
	assert.Equal(t, []string{"Sławomir Jasiński <slawomir@jasinski.us>"}, parsedEmail.Headers["From"])
}