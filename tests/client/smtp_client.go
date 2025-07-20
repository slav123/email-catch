package client

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPClient struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
}

type EmailMessage struct {
	From        string
	To          []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

func NewSMTPClient(host string, port int) *SMTPClient {
	return &SMTPClient{
		host: host,
		port: port,
	}
}

func (c *SMTPClient) WithAuth(username, password string) *SMTPClient {
	c.username = username
	c.password = password
	return c
}

func (c *SMTPClient) WithTLS(useTLS bool) *SMTPClient {
	c.useTLS = useTLS
	return c
}

func (c *SMTPClient) SendEmail(message EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var conn *smtp.Client
	var err error

	if c.useTLS && (c.port == 465 || c.port == 993) {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         c.host,
		}
		
		tlsConn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}

		conn, err = smtp.NewClient(tlsConn, c.host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		conn, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}

		if c.useTLS {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         c.host,
			}
			if err := conn.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	defer conn.Quit()

	if c.username != "" && c.password != "" {
		auth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	if err := conn.Mail(message.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range message.To {
		if err := conn.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	writer, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	emailContent := c.buildEmailContent(message)
	if _, err := writer.Write([]byte(emailContent)); err != nil {
		return fmt.Errorf("failed to write email content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

func (c *SMTPClient) buildEmailContent(message EmailMessage) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("From: %s\r\n", message.From))
	content.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(message.To, ", ")))
	content.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))
	content.WriteString("MIME-Version: 1.0\r\n")

	if len(message.Attachments) > 0 {
		boundary := "----=_Part_0_123456789.123456789"
		content.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		content.WriteString("\r\n")

		content.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if message.HTMLBody != "" {
			content.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			content.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		content.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		content.WriteString("\r\n")
		
		if message.HTMLBody != "" {
			content.WriteString(message.HTMLBody)
		} else {
			content.WriteString(message.Body)
		}
		content.WriteString("\r\n")

		for _, attachment := range message.Attachments {
			content.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			content.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
			content.WriteString("Content-Transfer-Encoding: base64\r\n")
			content.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
			content.WriteString("\r\n")
			content.WriteString(c.encodeBase64(attachment.Content))
			content.WriteString("\r\n")
		}

		content.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		if message.HTMLBody != "" {
			content.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			content.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		content.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		content.WriteString("\r\n")
		
		if message.HTMLBody != "" {
			content.WriteString(message.HTMLBody)
		} else {
			content.WriteString(message.Body)
		}
	}

	return content.String()
}

func (c *SMTPClient) encodeBase64(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	
	for i := 0; i < len(data); i += 3 {
		chunk := data[i:min(i+3, len(data))]
		
		var val uint32
		for j, b := range chunk {
			val |= uint32(b) << (16 - 8*j)
		}
		
		for j := 0; j < 4; j++ {
			if i*4/3+j < (len(data)+2)/3*4 {
				if len(chunk) > j*3/4 {
					result.WriteByte(chars[(val>>(18-6*j))&0x3F])
				} else {
					result.WriteByte('=')
				}
			}
		}
		
		if result.Len()%76 == 0 {
			result.WriteString("\r\n")
		}
	}
	
	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}