package email

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// MarkdownConverter handles conversion of email content to markdown
type MarkdownConverter struct {
	baseURL    string
	folderPath string
}

// NewMarkdownConverter creates a new markdown converter with the specified base URL and folder path
func NewMarkdownConverter(baseURL, folderPath string) *MarkdownConverter {
	return &MarkdownConverter{
		baseURL:    baseURL,
		folderPath: folderPath,
	}
}

// ConvertToMarkdown converts an email to markdown format
func (mc *MarkdownConverter) ConvertToMarkdown(email *Email) string {
	var markdown strings.Builder

	// Email headers
	markdown.WriteString("# Email\n\n")
	markdown.WriteString(fmt.Sprintf("**From:** %s\n", email.From))
	markdown.WriteString(fmt.Sprintf("**To:** %s\n", strings.Join(email.To, ", ")))
	markdown.WriteString(fmt.Sprintf("**Subject:** %s\n", email.Subject))
	markdown.WriteString(fmt.Sprintf("**Date:** %s\n", email.Date.Format("2006-01-02 15:04:05")))
	if email.MessageID != "" {
		markdown.WriteString(fmt.Sprintf("**Message-ID:** %s\n", email.MessageID))
	}
	markdown.WriteString("\n---\n\n")

	// Email body content
	if email.HTMLBody != "" {
		// Convert HTML to markdown
		markdownBody := mc.convertHTMLToMarkdown(email.HTMLBody, email.Attachments)
		markdown.WriteString(markdownBody)
	} else if email.Body != "" {
		// Use plain text body
		markdown.WriteString(email.Body)
	}

	// Add attachments section if there are any non-image attachments
	nonImageAttachments := mc.getNonImageAttachments(email.Attachments)
	if len(nonImageAttachments) > 0 {
		markdown.WriteString("\n\n## Attachments\n\n")
		for _, attachment := range nonImageAttachments {
			attachmentURL := mc.getAttachmentURL(attachment.Filename)
			markdown.WriteString(fmt.Sprintf("- [%s](%s) (%s, %s)\n",
				attachment.Filename,
				attachmentURL,
				attachment.ContentType,
				mc.formatFileSize(attachment.Size)))
		}
	}

	return markdown.String()
}

// convertHTMLToMarkdown converts HTML content to markdown, handling images specially
func (mc *MarkdownConverter) convertHTMLToMarkdown(htmlContent string, attachments []Attachment) string {
	// Simple HTML to markdown conversion
	content := htmlContent

	// Handle images first - replace img tags with markdown images
	content = mc.replaceImagesWithMarkdown(content, attachments)

	// Basic HTML tag conversions
	content = mc.convertBasicHTMLTags(content)

	// Clean up extra whitespace
	content = mc.cleanupWhitespace(content)

	return content
}

// replaceImagesWithMarkdown replaces HTML img tags with markdown image syntax
func (mc *MarkdownConverter) replaceImagesWithMarkdown(content string, attachments []Attachment) string {
	// Find all img tags
	imgRegex := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)

	content = imgRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract src attribute
		srcRegex := regexp.MustCompile(`src=["']([^"']+)["']`)
		srcMatch := srcRegex.FindStringSubmatch(match)
		if len(srcMatch) < 2 {
			return match
		}

		src := srcMatch[1]

		// Check if this is a reference to an attachment
		if attachment := mc.findAttachmentBySrc(src, attachments); attachment != nil {
			if mc.isImageAttachment(attachment) {
				imageURL := mc.getAttachmentURL(attachment.Filename)
				return fmt.Sprintf("![%s](%s)", attachment.Filename, imageURL)
			}
		}

		// If not found in attachments, keep original URL or try to make it absolute
		if strings.HasPrefix(src, "http") {
			return fmt.Sprintf("![Image](%s)", src)
		}

		return match // Keep original if we can't process it
	})

	return content
}

// convertBasicHTMLTags converts basic HTML tags to markdown
func (mc *MarkdownConverter) convertBasicHTMLTags(content string) string {
	// Headers
	content = regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`).ReplaceAllString(content, "# $1")
	content = regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`).ReplaceAllString(content, "## $1")
	content = regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`).ReplaceAllString(content, "### $1")
	content = regexp.MustCompile(`<h4[^>]*>(.*?)</h4>`).ReplaceAllString(content, "#### $1")
	content = regexp.MustCompile(`<h5[^>]*>(.*?)</h5>`).ReplaceAllString(content, "##### $1")
	content = regexp.MustCompile(`<h6[^>]*>(.*?)</h6>`).ReplaceAllString(content, "###### $1")

	// Bold and italic
	content = regexp.MustCompile(`<strong[^>]*>(.*?)</strong>`).ReplaceAllString(content, "**$1**")
	content = regexp.MustCompile(`<b[^>]*>(.*?)</b>`).ReplaceAllString(content, "**$1**")
	content = regexp.MustCompile(`<em[^>]*>(.*?)</em>`).ReplaceAllString(content, "*$1*")
	content = regexp.MustCompile(`<i[^>]*>(.*?)</i>`).ReplaceAllString(content, "*$1*")

	// Links
	content = regexp.MustCompile(`<a[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`).ReplaceAllString(content, "[$2]($1)")

	// Lists
	content = regexp.MustCompile(`<ul[^>]*>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`</ul>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<ol[^>]*>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`</ol>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<li[^>]*>(.*?)</li>`).ReplaceAllString(content, "- $1")

	// Paragraphs and line breaks
	content = regexp.MustCompile(`<p[^>]*>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`</p>`).ReplaceAllString(content, "\n\n")
	content = regexp.MustCompile(`<br[^>]*/?>`).ReplaceAllString(content, "\n")

	// Divs
	content = regexp.MustCompile(`<div[^>]*>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`</div>`).ReplaceAllString(content, "\n")

	// Remove remaining HTML tags
	content = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(content, "")

	// Decode HTML entities
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")

	return content
}

// cleanupWhitespace removes extra whitespace and normalizes line breaks
func (mc *MarkdownConverter) cleanupWhitespace(content string) string {
	// Remove excessive newlines
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	// Remove trailing whitespace on lines
	content = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(content, "\n")

	// Remove leading/trailing whitespace
	content = strings.TrimSpace(content)

	return content
}

// findAttachmentBySrc finds an attachment by its source reference
func (mc *MarkdownConverter) findAttachmentBySrc(src string, attachments []Attachment) *Attachment {
	// Try to match by filename
	for i := range attachments {
		if strings.Contains(src, attachments[i].Filename) {
			return &attachments[i]
		}
	}

	// Try to match by content-id (cid:)
	if strings.HasPrefix(src, "cid:") {
		cid := strings.TrimPrefix(src, "cid:")
		for i := range attachments {
			if strings.Contains(attachments[i].Filename, cid) {
				return &attachments[i]
			}
		}
	}

	return nil
}

// isImageAttachment checks if an attachment is an image
func (mc *MarkdownConverter) isImageAttachment(attachment *Attachment) bool {
	return strings.HasPrefix(attachment.ContentType, "image/")
}

// getNonImageAttachments returns attachments that are not images
func (mc *MarkdownConverter) getNonImageAttachments(attachments []Attachment) []Attachment {
	var nonImages []Attachment
	for _, attachment := range attachments {
		if !mc.isImageAttachment(&attachment) {
			nonImages = append(nonImages, attachment)
		}
	}
	return nonImages
}

// getAttachmentURL generates the full URL for an attachment
func (mc *MarkdownConverter) getAttachmentURL(filename string) string {
	// Parse the folder path to extract domain info
	parts := strings.Split(mc.folderPath, "/")
	if len(parts) == 0 {
		return fmt.Sprintf("%s/%s", mc.baseURL, filename)
	}

	// Extract folder name (first part) to determine subdomain
	folderName := parts[0]

	// Create subdomain URL
	var subdomain string
	switch folderName {
	case "komunikacja-pro":
		subdomain = "img.komunikacja.pro"
	case "faktury-hib":
		subdomain = "img.hib.pl"
	default:
		// Use base domain for unknown folders
		u, err := url.Parse(mc.baseURL)
		if err == nil {
			// Remove protocol from base URL to get clean domain
			if u.Host != "" {
				subdomain = "img." + u.Host
			} else {
				subdomain = "img.example.com"
			}
		} else {
			subdomain = "img.example.com"
		}
	}

	// Use path without the folder name (since it's encoded in subdomain)
	// Extract everything after the first part (folder name)
	pathWithoutFolder := strings.Join(parts[1:], "/")
	return fmt.Sprintf("https://%s/%s/%s", subdomain, pathWithoutFolder, filename)
}

// formatFileSize formats file size in human-readable format
func (mc *MarkdownConverter) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
