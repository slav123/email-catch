package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

type EmailPayload struct {
	From        string              `json:"from"`
	To          []string            `json:"to"`
	Subject     string              `json:"subject"`
	Date        time.Time           `json:"date"`
	MessageID   string              `json:"message_id"`
	Body        string              `json:"body"`
	HTMLBody    string              `json:"html_body"`
	Markdown    string              `json:"markdown,omitempty"`
	Headers     map[string][]string `json:"headers"`
	Attachments []AttachmentInfo    `json:"attachments"`
	Timestamp   time.Time           `json:"timestamp"`
	EMLPath     string              `json:"eml_path,omitempty"`
}

type AttachmentInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	S3Path      string `json:"s3_path,omitempty"`
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SendWebhook(url, method string, headers map[string]string, payload EmailPayload) error {
	payload.Timestamp = time.Now()

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "email-catch/2.0-markdown-compression")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) SendWebhookWithRetry(url, method string, headers map[string]string, payload EmailPayload, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*attempt) * time.Second
			time.Sleep(waitTime)
		}

		err := c.SendWebhook(url, method, headers, payload)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries+1, lastErr)
}