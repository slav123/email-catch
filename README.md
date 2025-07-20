# Email Catch

A high-performance Go-based email capture service that receives incoming emails via SMTP and processes them according to configurable routing rules. Supports multiple storage backends (S3-compatible and local filesystem) and webhook notifications.

## Features

- **Multi-Port SMTP Server**: Listen on multiple ports simultaneously (25, 587, 465, 2525)
- **TLS/SSL Support**: Secure email transmission with configurable TLS
- **Let's Encrypt Integration**: Automatic SSL certificate generation and renewal
- **S3-Compatible Storage**: Store emails using Minio SDK (works with AWS S3, MinIO, DigitalOcean Spaces, etc.)
- **Local Storage**: Save emails to local filesystem
- **Webhook Integration**: Send email notifications via HTTP webhooks
- **Routing Engine**: Route emails based on recipient patterns, sender patterns, and subject patterns
- **Email Parsing**: Full RFC822 email parsing with attachment support
- **Comprehensive Testing**: Unit tests, integration tests, and load testing tools

## Quick Start

### Using Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/slav123/email-catch.git
cd email-catch
```

2. Start the services:
```bash
docker-compose up -d
```

3. The service will be available on:
   - SMTP: ports 25, 587, 465, 2525
   - MinIO Console: http://localhost:9001 (admin/admin123)

### Using Go directly

1. Install dependencies:
```bash
go mod tidy
```

2. Start MinIO (optional, for S3 storage):
```bash
make dev-minio
```

3. Run the service:
```bash
make run
```

## Configuration

The service is configured via YAML files. See `config/config.yaml` for the main configuration and `config/test-config.yaml` for testing configuration.

### Example Configuration

```yaml
server:
  ports: [25, 587, 465, 2525]
  hostname: "mail.example.com"
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    letsencrypt:
      enabled: true
      domains: ["mail.example.com"]
      email: "admin@example.com"
      staging: false
      http_port: 80

storage:
  s3_compatible:
    enabled: true
    endpoint: "localhost:9000"
    access_key: "minioadmin"
    secret_key: "minioadmin"
    bucket: "email-catch"
  local:
    enabled: true
    directory: "./emails"

routes:
  - name: "capture_all"
    condition:
      recipient_pattern: "capture@.*"
    actions:
      - type: "store_s3"
        enabled: true
      - type: "webhook"
        enabled: true
        config:
          url: "http://localhost:8080/webhook"
    enabled: true
```

## Routing Rules

The routing engine supports pattern matching on:

- **Recipient Pattern**: Match against email recipients
- **Sender Pattern**: Match against email sender
- **Subject Pattern**: Match against email subject

### Available Actions

- **store_local**: Save email to local filesystem
- **store_s3**: Upload email to S3-compatible storage
- **webhook**: Send email data via HTTP POST

## Testing

### Run All Tests

```bash
make test
```

### Run Specific Test Types

```bash
# Unit tests only
make test-unit

# Integration tests only
make test-integration

# Docker-based tests
make docker-test
```

### Send Test Emails

```bash
# Send simple test email
make test-send-simple

# Send email with attachments
make test-send-attachments

# Test all ports
make test-send-all-ports

# Load testing
make test-send-load
```

### Custom Test Email Sending

```bash
# Send to specific ports
go run ./tests/scripts/test_email_sender.go -ports "25,587" -count 5

# Send different email types
go run ./tests/scripts/test_email_sender.go -type attachments -count 3

# Send with TLS
go run ./tests/scripts/test_email_sender.go -tls -ports "465,587"
```

## Email Types Supported

The service can handle various email types:

- **Simple Text**: Plain text emails
- **HTML**: HTML emails with alternative text versions
- **Attachments**: Emails with binary attachments (images, documents, etc.)
- **Large Emails**: Emails with large attachments (>10MB)
- **Multi-recipient**: Emails sent to multiple recipients
- **Special Characters**: Unicode and international characters
- **Complex MIME**: Nested multipart messages

## Storage Backends

### S3-Compatible Storage

Supports any S3-compatible storage service:
- Amazon S3
- MinIO
- DigitalOcean Spaces
- Wasabi
- Backblaze B2

### Local Storage

Emails are saved as `.eml` files in the configured directory with the following naming convention:
```
YYYYMMDD_HHMMSS_<message-id>.eml
```

## Webhook Format

When webhook actions are triggered, the service sends a JSON payload:

```json
{
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "subject": "Email Subject",
  "date": "2023-10-15T10:30:00Z",
  "message_id": "<123@example.com>",
  "body": "Plain text body",
  "html_body": "<html>HTML body</html>",
  "headers": {
    "Content-Type": ["text/plain"],
    "X-Custom": ["value"]
  },
  "attachments": [
    {
      "filename": "document.pdf",
      "content_type": "application/pdf",
      "size": 12345
    }
  ],
  "timestamp": "2023-10-15T10:30:01Z"
}
```

## Development

### Project Structure

```
email-catch/
├── cmd/server/          # Main application
├── internal/
│   ├── config/         # Configuration management
│   ├── smtp/           # SMTP server implementation
│   ├── storage/        # Storage backends
│   └── webhook/        # Webhook client
├── pkg/email/          # Email parsing and processing
├── tests/
│   ├── unit/           # Unit tests
│   ├── integration/    # Integration tests
│   ├── client/         # Test SMTP client
│   ├── testdata/       # Test email templates
│   └── scripts/        # Test automation scripts
└── config/             # Configuration files
```

### Building

```bash
make build
```

### Development Helpers

```bash
# Set up development environment
make dev-setup

# Start MinIO for development
make dev-minio

# Generate TLS certificates
make gen-certs

# Format code
make fmt

# Run linter
make lint
```

## Let's Encrypt SSL Certificates

Email Catch includes built-in Let's Encrypt support for automatic SSL certificate generation and renewal.

### Configuration

```yaml
server:
  tls:
    enabled: true
    letsencrypt:
      enabled: true
      domains: ["mail.yourdomain.com"]
      email: "admin@yourdomain.com"
      staging: false  # Set to true for testing
      http_port: 80   # Port for HTTP-01 challenge
      renew_before_days: 30
```

### Certificate Management

```bash
# Get certificate information
./bin/certmanager -action info

# Manually renew certificates
./bin/certmanager -action renew

# Generate self-signed certificates for testing
./bin/certmanager -action generate-self-signed -domains "localhost"
```

### Production Deployment

See `deploy/DEPLOYMENT.md` for complete production deployment instructions with Let's Encrypt.

### Requirements

- Public domain name pointing to your server
- Port 80 open for HTTP-01 challenge verification
- Valid email address for Let's Encrypt registration

## Security Considerations

- Always use TLS in production environments
- Implement proper firewall rules for SMTP ports
- Use strong authentication for S3 storage
- Validate webhook endpoints
- Monitor for abuse and implement rate limiting
- Let's Encrypt certificates are automatically renewed every 30 days

## Performance

The service is designed for high performance:
- Concurrent connection handling
- Efficient email parsing
- Streaming S3 uploads
- Configurable rate limiting
- Graceful shutdown handling

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Support

For issues and feature requests, please use the GitHub issue tracker.