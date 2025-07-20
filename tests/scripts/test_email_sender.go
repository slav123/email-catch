package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/slav123/email-catch/tests/client"
	"github.com/slav123/email-catch/tests/testdata"
)

func main() {
	var (
		host       = flag.String("host", "localhost", "SMTP server host")
		ports      = flag.String("ports", "25,587,465,2525", "Comma-separated list of ports to test")
		recipient  = flag.String("to", "capture@example.com", "Recipient email address")
		sender     = flag.String("from", "test@example.com", "Sender email address")
		count      = flag.Int("count", 1, "Number of emails to send per port")
		withTLS    = flag.Bool("tls", false, "Use TLS connection")
		emailType  = flag.String("type", "all", "Type of email to send: simple, html, attachments, large, multi, special, long, all")
		delay      = flag.Duration("delay", 100*time.Millisecond, "Delay between emails")
	)
	flag.Parse()

	portList := strings.Split(*ports, ",")
	var portNumbers []int
	for _, portStr := range portList {
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			log.Fatalf("Invalid port number: %s", portStr)
		}
		portNumbers = append(portNumbers, port)
	}

	var emailTemplates []testdata.EmailTemplate
	switch *emailType {
	case "simple":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateSimpleEmail()}
	case "html":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateHTMLEmail()}
	case "attachments":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateEmailWithAttachments()}
	case "large":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateLargeEmail()}
	case "multi":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateMultiRecipientEmail()}
	case "special":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateEmailWithSpecialCharacters()}
	case "long":
		emailTemplates = []testdata.EmailTemplate{testdata.GenerateEmailWithLongSubject()}
	case "all":
		emailTemplates = testdata.GetAllTestEmails()
	default:
		log.Fatalf("Unknown email type: %s", *emailType)
	}

	fmt.Printf("Sending %d emails to %d ports using %d email templates\n", *count, len(portNumbers), len(emailTemplates))
	fmt.Printf("Ports: %v\n", portNumbers)
	fmt.Printf("Email types: %s\n", *emailType)
	fmt.Printf("TLS: %v\n", *withTLS)
	fmt.Printf("Delay: %v\n", *delay)

	totalEmails := 0
	successCount := 0
	failureCount := 0

	for _, port := range portNumbers {
		fmt.Printf("\n--- Testing port %d ---\n", port)
		
		smtpClient := client.NewSMTPClient(*host, port).WithTLS(*withTLS)

		for templateIndex, template := range emailTemplates {
			for i := 0; i < *count; i++ {
				message := client.EmailMessage{
					From:        *sender,
					To:          []string{*recipient},
					Subject:     fmt.Sprintf("[Port %d] %s (#%d)", port, template.Subject, i+1),
					Body:        template.Body,
					HTMLBody:    template.HTMLBody,
					Attachments: convertAttachments(template.Attachments),
				}

				fmt.Printf("Sending email %d/%d (template %d/%d) to port %d... ", 
					i+1, *count, templateIndex+1, len(emailTemplates), port)

				err := smtpClient.SendEmail(message)
				if err != nil {
					fmt.Printf("FAILED: %v\n", err)
					failureCount++
				} else {
					fmt.Printf("SUCCESS\n")
					successCount++
				}

				totalEmails++

				if *delay > 0 {
					time.Sleep(*delay)
				}
			}
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Total emails sent: %d\n", totalEmails)
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failureCount)
	fmt.Printf("Success rate: %.2f%%\n", float64(successCount)/float64(totalEmails)*100)
}

func convertAttachments(templates []testdata.AttachmentTemplate) []client.Attachment {
	var attachments []client.Attachment
	for _, template := range templates {
		attachments = append(attachments, client.Attachment{
			Filename:    template.Filename,
			ContentType: template.ContentType,
			Content:     template.Content,
		})
	}
	return attachments
}