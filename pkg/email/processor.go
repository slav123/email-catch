package email

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/slav123/email-catch/internal/config"
	"github.com/slav123/email-catch/internal/storage"
	"github.com/slav123/email-catch/internal/webhook"
)

type Processor struct {
	config         *config.Config
	storageBackend storage.Backend
	webhookClient  *webhook.Client
}

func NewProcessor(cfg *config.Config, storageBackend storage.Backend, webhookClient *webhook.Client) *Processor {
	return &Processor{
		config:         cfg,
		storageBackend: storageBackend,
		webhookClient:  webhookClient,
	}
}

func (p *Processor) ProcessEmail(from string, to []string, rawData []byte) error {
	email, err := ParseEmail(rawData, from, to)
	if err != nil {
		return fmt.Errorf("failed to parse email: %w", err)
	}

	log.Printf("Processing email: %s", email.Summary())

	routes := p.config.GetEnabledRoutes()
	matchedRoutes := p.findMatchingRoutes(email, routes)

	if len(matchedRoutes) == 0 {
		log.Printf("No matching routes for email from %s to %v", from, to)
		return nil
	}

	for _, route := range matchedRoutes {
		if err := p.executeRoute(email, route); err != nil {
			log.Printf("Failed to execute route %s: %v", route.Name, err)
			continue
		}
		log.Printf("Successfully executed route: %s", route.Name)
	}

	return nil
}

func (p *Processor) findMatchingRoutes(email *Email, routes []config.RouteConfig) []config.RouteConfig {
	var matched []config.RouteConfig

	for _, route := range routes {
		if p.routeMatches(email, route) {
			matched = append(matched, route)
		}
	}

	return matched
}

func (p *Processor) routeMatches(email *Email, route config.RouteConfig) bool {
	if route.Condition.RecipientPattern != "" {
		matched := false
		pattern, err := regexp.Compile(route.Condition.RecipientPattern)
		if err != nil {
			log.Printf("Invalid recipient pattern in route %s: %v", route.Name, err)
			return false
		}

		for _, recipient := range email.To {
			if pattern.MatchString(recipient) {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	if route.Condition.SenderPattern != "" {
		pattern, err := regexp.Compile(route.Condition.SenderPattern)
		if err != nil {
			log.Printf("Invalid sender pattern in route %s: %v", route.Name, err)
			return false
		}

		if !pattern.MatchString(email.From) {
			return false
		}
	}

	if route.Condition.SubjectPattern != "" {
		pattern, err := regexp.Compile(route.Condition.SubjectPattern)
		if err != nil {
			log.Printf("Invalid subject pattern in route %s: %v", route.Name, err)
			return false
		}

		if !pattern.MatchString(email.Subject) {
			return false
		}
	}

	return true
}

func (p *Processor) executeRoute(email *Email, route config.RouteConfig) error {
	for _, action := range route.Actions {
		if !action.Enabled {
			continue
		}

		switch action.Type {
		case "store_local":
			if err := p.executeLocalStorage(email, action); err != nil {
				return fmt.Errorf("local storage action failed: %w", err)
			}
		case "store_s3":
			if err := p.executeS3Storage(email, action); err != nil {
				return fmt.Errorf("S3 storage action failed: %w", err)
			}
		case "webhook":
			if err := p.executeWebhook(email, action); err != nil {
				return fmt.Errorf("webhook action failed: %w", err)
			}
		default:
			log.Printf("Unknown action type: %s", action.Type)
		}
	}

	return nil
}

func (p *Processor) executeLocalStorage(email *Email, action config.Action) error {
	folder := action.Config["folder"]
	if folder == "" {
		folder = "default"
	}

	filename := p.generateFilename(email)
	uniqueID := p.generateUniqueID(email)
	// Add year and month to the path
	year := email.Date.Format("2006")
	month := email.Date.Format("01")
	
	// Create folder path using timestamp-based unique ID
	folderPath := fmt.Sprintf("%s/%s/%s/%s", folder, year, month, uniqueID)
	
	// Store the EML file
	emlPath := fmt.Sprintf("%s/%s", folderPath, filename)
	if err := p.storageBackend.StoreLocal(emlPath, email.ToEML()); err != nil {
		return fmt.Errorf("failed to store EML file: %w", err)
	}
	
	// Store attachments as separate files
	for _, attachment := range email.Attachments {
		attachmentPath := fmt.Sprintf("%s/%s", folderPath, attachment.Filename)
		if err := p.storageBackend.StoreLocal(attachmentPath, attachment.Content); err != nil {
			log.Printf("Failed to store attachment %s: %v", attachment.Filename, err)
			continue
		}
	}
	
	// Store webhook payload as JSON file
	if err := p.storeWebhookPayload(email, folderPath); err != nil {
		log.Printf("Failed to store webhook payload: %v", err)
	}
	
	return nil
}

func (p *Processor) executeS3Storage(email *Email, action config.Action) error {
	folder := action.Config["folder"]
	if folder == "" {
		folder = "default"
	}

	filename := p.generateFilename(email)
	uniqueID := p.generateUniqueID(email)
	// Add year and month to the path
	year := email.Date.Format("2006")
	month := email.Date.Format("01")
	
	// Create folder path using timestamp-based unique ID
	folderPath := fmt.Sprintf("%s/%s/%s/%s", folder, year, month, uniqueID)
	
	// Store the EML file
	emlPath := fmt.Sprintf("%s/%s", folderPath, filename)
	if err := p.storageBackend.StoreS3(emlPath, email.ToEML()); err != nil {
		return fmt.Errorf("failed to store EML file: %w", err)
	}
	
	// Store attachments as separate files
	for _, attachment := range email.Attachments {
		attachmentPath := fmt.Sprintf("%s/%s", folderPath, attachment.Filename)
		if err := p.storageBackend.StoreS3WithContentType(attachmentPath, attachment.Content, attachment.ContentType); err != nil {
			log.Printf("Failed to store attachment %s: %v", attachment.Filename, err)
			continue
		}
	}
	
	// Store webhook payload as JSON file
	if err := p.storeWebhookPayload(email, folderPath); err != nil {
		log.Printf("Failed to store webhook payload: %v", err)
	}
	
	return nil
}

func (p *Processor) executeWebhook(email *Email, action config.Action) error {
	url := action.Config["url"]
	if url == "" {
		return fmt.Errorf("webhook URL not specified")
	}

	method := action.Config["method"]
	if method == "" {
		method = "POST"
	}

	headers := make(map[string]string)
	if headerStr := action.Config["headers"]; headerStr != "" {
		pairs := strings.Split(headerStr, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, ":", 2); len(kv) == 2 {
				headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	// Generate folder structure for paths - use current route's folder or find S3 folder
	folder := p.getRouteFolder(email)
	
	filename := p.generateFilename(email)
	uniqueID := p.generateUniqueID(email)
	year := email.Date.Format("2006")
	month := email.Date.Format("01")
	folderPath := fmt.Sprintf("%s/%s/%s/%s", folder, year, month, uniqueID)
	
	// Generate markdown version of the email
	markdownConverter := NewMarkdownConverter("https://img.example.com", folderPath)
	markdownContent := markdownConverter.ConvertToMarkdown(email)
	
	payload := webhook.EmailPayload{
		From:        email.From,
		To:          email.To,
		Subject:     email.Subject,
		Date:        email.Date,
		MessageID:   email.MessageID,
		Body:        email.Body,
		HTMLBody:    email.HTMLBody,
		Markdown:    markdownContent,
		Headers:     email.Headers,
		Attachments: make([]webhook.AttachmentInfo, len(email.Attachments)),
		EMLPath:     fmt.Sprintf("%s/%s", folderPath, filename),
	}

	for i, att := range email.Attachments {
		payload.Attachments[i] = webhook.AttachmentInfo{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			S3Path:      fmt.Sprintf("%s/%s", folderPath, att.Filename),
		}
	}

	return p.webhookClient.SendWebhook(url, method, headers, payload)
}

func (p *Processor) generateUniqueID(email *Email) string {
	// Use microseconds for better uniqueness to avoid collisions
	timestamp := email.Date.Format("20060102_150405.000000")
	
	// If there's a MessageID, include a short hash for extra uniqueness
	if email.MessageID != "" {
		// Create a short hash from MessageID to ensure uniqueness
		hash := fmt.Sprintf("%x", []byte(email.MessageID))
		if len(hash) > 8 {
			hash = hash[:8]
		}
		return fmt.Sprintf("%s_%s", timestamp, hash)
	}
	
	return timestamp
}

func (p *Processor) generateFilename(email *Email) string {
	timestamp := email.Date.Format("20060102_150405")
	messageID := email.MessageID
	if messageID == "" {
		messageID = "no-id"
	}

	messageID = strings.ReplaceAll(messageID, "<", "")
	messageID = strings.ReplaceAll(messageID, ">", "")
	messageID = strings.ReplaceAll(messageID, "@", "_at_")
	messageID = strings.ReplaceAll(messageID, "/", "_")
	messageID = strings.ReplaceAll(messageID, "\\", "_")

	if len(messageID) > 50 {
		messageID = messageID[:50]
	}

	return fmt.Sprintf("%s_%s.eml", timestamp, messageID)
}

// GenerateFilename generates a filename for an email (public method for testing)
func (p *Processor) GenerateFilename(email *Email) string {
	return p.generateFilename(email)
}

// GetRouteFolder gets the folder name from the current route actions (public method for testing)
func (p *Processor) GetRouteFolder(email *Email) string {
	return p.getRouteFolder(email)
}

// findS3StorageAction looks for an S3 storage action in the current routes
func (p *Processor) findS3StorageAction(email *Email) *config.Action {
	routes := p.config.GetEnabledRoutes()
	matchedRoutes := p.findMatchingRoutes(email, routes)
	
	for _, route := range matchedRoutes {
		for _, action := range route.Actions {
			if action.Type == "store_s3" && action.Enabled {
				return &action
			}
		}
	}
	return nil
}

// getRouteFolder gets the folder name from the current route actions
func (p *Processor) getRouteFolder(email *Email) string {
	routes := p.config.GetEnabledRoutes()
	matchedRoutes := p.findMatchingRoutes(email, routes)
	
	for _, route := range matchedRoutes {
		for _, action := range route.Actions {
			if action.Enabled && action.Config["folder"] != "" {
				return action.Config["folder"]
			}
		}
	}
	
	return "default"
}

// storeWebhookPayload creates and stores the webhook payload as a JSON file
func (p *Processor) storeWebhookPayload(email *Email, folderPath string) error {
	// Use the folderPath that was passed in - it's already correctly constructed
	filename := p.generateFilename(email)
	baseFilename := strings.TrimSuffix(filename, ".eml")
	
	// Generate markdown version of the email
	markdownConverter := NewMarkdownConverter("https://img.example.com", folderPath)
	markdownContent := markdownConverter.ConvertToMarkdown(email)
	
	payload := webhook.EmailPayload{
		From:        email.From,
		To:          email.To,
		Subject:     email.Subject,
		Date:        email.Date,
		MessageID:   email.MessageID,
		Body:        email.Body,
		HTMLBody:    email.HTMLBody,
		Markdown:    markdownContent,
		Headers:     email.Headers,
		Attachments: make([]webhook.AttachmentInfo, len(email.Attachments)),
		EMLPath:     fmt.Sprintf("%s/%s", folderPath, filename),
	}

	for i, att := range email.Attachments {
		payload.Attachments[i] = webhook.AttachmentInfo{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			S3Path:      fmt.Sprintf("%s/%s", folderPath, att.Filename),
		}
	}
	
	// Marshal payload to JSON
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}
	
	// Store JSON file
	jsonPath := fmt.Sprintf("%s/%s.json", folderPath, baseFilename)
	
	// Try S3 storage first if enabled
	if p.config.Storage.S3Compatible.Enabled {
		if err := p.storageBackend.StoreS3WithContentType(jsonPath, jsonData, "application/json"); err != nil {
			return fmt.Errorf("failed to store JSON to S3: %w", err)
		}
	}
	
	// Store locally if enabled
	if p.config.Storage.Local.Enabled {
		if err := p.storageBackend.StoreLocal(jsonPath, jsonData); err != nil {
			return fmt.Errorf("failed to store JSON locally: %w", err)
		}
	}
	
	return nil
}