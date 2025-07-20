package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

type Email struct {
	From        string
	To          []string
	Subject     string
	Date        time.Time
	MessageID   string
	Headers     map[string][]string
	Body        string
	HTMLBody    string
	Attachments []Attachment
	Raw         []byte
}

type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	Size        int64
}

func ParseEmail(rawData []byte, from string, to []string) (*Email, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	email := &Email{
		From:        from,
		To:          to,
		Headers:     make(map[string][]string),
		Raw:         rawData,
		Attachments: make([]Attachment, 0),
	}

	for key, values := range msg.Header {
		decodedValues := make([]string, len(values))
		for i, value := range values {
			decodedValues[i] = decodeMIMEHeader(value)
		}
		email.Headers[key] = decodedValues
	}

	email.Subject = decodeMIMEHeader(msg.Header.Get("Subject"))
	email.MessageID = decodeMIMEHeader(msg.Header.Get("Message-Id"))
	
	// Also decode the From header if present (may be different from SMTP envelope)
	if fromHeader := msg.Header.Get("From"); fromHeader != "" {
		email.From = decodeMIMEHeader(fromHeader)
	}

	if dateStr := msg.Header.Get("Date"); dateStr != "" {
		if parsedDate, err := mail.ParseDate(dateStr); err == nil {
			email.Date = parsedDate
		} else {
			email.Date = time.Now()
		}
	} else {
		email.Date = time.Now()
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content type: %w", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("missing boundary in multipart message")
		}

		if err := email.parseMultipart(msg.Body, boundary); err != nil {
			return nil, fmt.Errorf("failed to parse multipart message: %w", err)
		}
	} else {
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read message body: %w", err)
		}

		encoding := msg.Header.Get("Content-Transfer-Encoding")
		decoded, err := decodeContent(body, encoding)
		if err != nil {
			return nil, fmt.Errorf("failed to decode message body: %w", err)
		}

		if strings.HasPrefix(mediaType, "text/html") {
			email.HTMLBody = string(decoded)
		} else {
			email.Body = string(decoded)
		}
	}

	return email, nil
}

func (e *Email) parseMultipart(body io.Reader, boundary string) error {
	reader := multipart.NewReader(body, boundary)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read multipart: %w", err)
		}

		contentType := part.Header.Get("Content-Type")
		contentDisposition := part.Header.Get("Content-Disposition")
		contentTransferEncoding := part.Header.Get("Content-Transfer-Encoding")

		partData, err := io.ReadAll(part)
		if err != nil {
			return fmt.Errorf("failed to read part data: %w", err)
		}

		decoded, err := decodeContent(partData, contentTransferEncoding)
		if err != nil {
			return fmt.Errorf("failed to decode part content: %w", err)
		}

		disposition, params, _ := mime.ParseMediaType(contentDisposition)
		
		// Debug: log part information
		log.Printf("DEBUG: Part - ContentType: %s, ContentDisposition: %s, Size: %d, Disposition: %s", 
			contentType, contentDisposition, len(decoded), disposition)

		// Check if this is an attachment based on various indicators
		isAttachment := false
		filename := ""
		
		// Standard attachment detection
		if disposition == "attachment" || (disposition == "" && params["filename"] != "") {
			isAttachment = true
			filename = params["filename"]
		}
		
		// Check for application/* content types that are likely attachments
		mediaType, mediaParams, _ := mime.ParseMediaType(contentType)
		if strings.HasPrefix(mediaType, "application/") && !strings.HasPrefix(mediaType, "application/text") {
			isAttachment = true
			if filename == "" {
				filename = mediaParams["name"]
			}
		}
		
		// Check for image attachments
		if strings.HasPrefix(mediaType, "image/") {
			isAttachment = true
			if filename == "" {
				filename = mediaParams["name"]
			}
		}
		
		// Handle message/rfc822 as forwarded email attachment
		if strings.HasPrefix(contentType, "message/rfc822") || strings.Contains(contentType, "forwarded-message") {
			isAttachment = true
			if filename == "" {
				filename = "forwarded_email.eml"
			}
		}
		
		if isAttachment {
			if filename == "" {
				// Generate filename based on content type
				ext := getFileExtension(mediaType)
				filename = fmt.Sprintf("attachment_%d%s", len(e.Attachments)+1, ext)
			}
			
			log.Printf("DEBUG: Found attachment - Filename: %s, ContentType: %s, Size: %d", 
				filename, contentType, len(decoded))
			
			attachment := Attachment{
				Filename:    filename,
				ContentType: contentType,
				Content:     decoded,
				Size:        int64(len(decoded)),
			}
			e.Attachments = append(e.Attachments, attachment)
		} else {
			mediaType, _, _ := mime.ParseMediaType(contentType)

			if strings.HasPrefix(mediaType, "text/html") {
				e.HTMLBody = string(decoded)
			} else if strings.HasPrefix(mediaType, "text/plain") {
				e.Body = string(decoded)
			} else if strings.HasPrefix(mediaType, "multipart/") {
				nested := bytes.NewReader(decoded)
				nestedParams := make(map[string]string)
				if _, p, err := mime.ParseMediaType(contentType); err == nil {
					nestedParams = p
				}
				if boundary := nestedParams["boundary"]; boundary != "" {
					if err := e.parseMultipart(nested, boundary); err != nil {
						return fmt.Errorf("failed to parse nested multipart: %w", err)
					}
				}
			}
		}

		part.Close()
	}

	return nil
}

func decodeContent(data []byte, encoding string) ([]byte, error) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))

	switch encoding {
	case "base64":
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(decoded, data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		return decoded[:n], nil
	case "quoted-printable":
		reader := strings.NewReader(string(data))
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode quoted-printable: %w", err)
		}
		return decoded, nil
	case "7bit", "8bit", "binary", "":
		return data, nil
	default:
		return data, nil
	}
}

func (e *Email) ToEML() []byte {
	return e.Raw
}

func (e *Email) GetAttachmentByName(filename string) *Attachment {
	for i, attachment := range e.Attachments {
		if attachment.Filename == filename {
			return &e.Attachments[i]
		}
	}
	return nil
}

func (e *Email) HasAttachments() bool {
	return len(e.Attachments) > 0
}

func (e *Email) GetTotalSize() int64 {
	return int64(len(e.Raw))
}

func (e *Email) GetAttachmentsSize() int64 {
	var total int64
	for _, attachment := range e.Attachments {
		total += attachment.Size
	}
	return total
}

func (e *Email) Summary() string {
	return fmt.Sprintf("From: %s, To: %v, Subject: %s, Attachments: %d, Size: %d bytes",
		e.From, e.To, e.Subject, len(e.Attachments), len(e.Raw))
}

// decodeMIMEHeader decodes MIME encoded-word headers according to RFC 2047
func decodeMIMEHeader(header string) string {
	if header == "" {
		return ""
	}
	
	// Use Go's built-in MIME word decoder
	decoder := &mime.WordDecoder{}
	decoded, err := decoder.DecodeHeader(header)
	if err != nil {
		// If decoding fails, return the original header
		return header
	}
	
	return decoded
}

// getFileExtension returns appropriate file extension based on media type
func getFileExtension(mediaType string) string {
	switch mediaType {
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/zip":
		return ".zip"
	case "application/x-rar-compressed":
		return ".rar"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	default:
		return ".bin"
	}
}