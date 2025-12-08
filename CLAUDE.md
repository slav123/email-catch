# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Essential Commands
- `make build` - Build both main server and certificate manager binaries
- `make run` - Run server with default config (`config/config.yaml`)
- `make run-test` - Run server with test config for development
- `make test` - Run all tests (unit + integration)
- `make test-unit` - Run only unit tests in `tests/unit/`
- `make test-integration` - Run integration tests
- `make clean` - Clean build artifacts and test directories

### Development Environment
- `make dev-setup` - Create required directories (`emails`, `logs`, `certs`)
- `make dev-minio` - Start MinIO container for S3-compatible storage testing
- `make docker-run` - Start full stack with docker-compose (includes MinIO)
- `make fmt` - Format Go code
- `make lint` - Run golangci-lint (if available)

### Testing Email Functionality
- `make test-send-simple` - Send a simple test email
- `make test-send-attachments` - Send email with attachments
- `make test-send-all-ports` - Test all configured SMTP ports
- `make test-send-load` - Send multiple emails for load testing
- `go run ./tests/scripts/test_email_sender.go -ports "25,587" -count 5` - Custom test email sending

### Certificate Management
- `make gen-certs` - Generate self-signed certificates for development
- `make cert-info` - Get Let's Encrypt certificate information
- `make cert-renew` - Manually renew Let's Encrypt certificates

### Docker Operations
- `make docker-build` - Build Docker image
- `make docker-test` - Run tests in Docker environment
- `make docker-stop` - Stop all Docker services

## Architecture Overview

### Core Components

**Email Flow**: Raw SMTP → Parser → Processor → Storage Backends + Webhooks

1. **SMTP Server** (`internal/smtp/server.go`)
   - Multi-port SMTP server with TLS support
   - Handles concurrent connections and enforces rate limiting
   - RFC 5321 compliant with multi-line EHLO responses

2. **Email Parser** (`pkg/email/parser.go`)
   - RFC 822/2047 compliant email parsing
   - MIME multipart handling with attachment extraction
   - Supports nested multipart messages and forwarded emails
   - MIME header decoding for international characters

3. **Email Processor** (`pkg/email/processor.go`)
   - Routes emails based on regex patterns (recipient, sender, subject)
   - Executes configurable actions: storage (S3/local), webhooks
   - Generates markdown versions of emails with image embedding
   - Creates organized folder structures: `folder/year/month/message-id/`

4. **Storage Backends** (`internal/storage/backend.go`)
   - S3-compatible storage (AWS S3, MinIO, etc.) with automatic gzip compression for text files
   - Local filesystem storage with same folder structure
   - Stores EML files, JSON metadata, and individual attachments
   - Compresses .eml and .json files to save significant storage space (70-90% reduction)

5. **Configuration System** (`internal/config/config.go`)
   - YAML-based configuration with validation
   - Routing rules engine for pattern matching
   - Support for multiple storage backends and webhook endpoints

### Key Features

**Email Processing Pipeline**:
- Emails are parsed into structured data with attachments extracted
- Routing engine matches emails against configurable patterns
- Actions execute in parallel: storage + webhooks + markdown generation
- Automatic gzip compression for text files (.eml, .json) to reduce S3 storage costs by 70-90%
- Binary attachments (PDFs, images) stored uncompressed as they're already optimized

**Storage Organization**:
```
folder/year/month/timestamp_message-id/
├── timestamp_message-id.eml (gzipped)
├── timestamp_message-id.json (gzipped, webhook payload)
├── attachment1.pdf
└── attachment2.jpg
```

**Webhook Integration**:
- Sends JSON payload with email metadata, markdown version, and S3 paths
- Includes attachment information with S3 paths for direct access
- User-Agent header includes version: `email-catch/2.0-markdown-compression`

**Markdown Generation** (`pkg/email/markdown.go`):
- Converts HTML emails to markdown with image embedding
- Domain-specific URL generation for attachments
- Handles complex HTML structures and MIME content

### Configuration Structure

Routes define email processing rules:
```yaml
routes:
  - name: "route_name"
    condition:
      recipient_pattern: "regex_pattern"
      sender_pattern: "regex_pattern" 
      subject_pattern: "regex_pattern"
    actions:
      - type: "store_s3|store_local|webhook"
        enabled: true
        config:
          folder: "folder_name"  # for storage
          url: "webhook_url"     # for webhooks
```

### Testing Architecture

- **Unit Tests** (`tests/unit/`): Test individual components (parser, processor, storage)
- **Integration Tests** (`tests/integration/`): End-to-end email flow testing
- **Test Data Generation** (`tests/testdata/`): Email templates for various scenarios
- **SMTP Test Client** (`tests/client/`): Custom SMTP client for testing

### Development vs Production

**Development**: Use `config/test-config.yaml` with local storage and test ports (2525-2528)
**Production**: Use `config/config.yaml` with S3 storage, standard SMTP ports (25, 587, 465), and Let's Encrypt TLS

### Deployment Notes

- Server typically runs as systemd service under dedicated `email-catch` user
- Requires port 80 open for Let's Encrypt HTTP-01 challenges
- Production deployment instructions in `deploy/DEPLOYMENT.md`
- Configuration files can be environment-specific (production, staging)

### Important Implementation Details

- Email parsing handles forwarded emails by detecting `message/rfc822` content
- Attachment detection works with `Content-Disposition: attachment` and content type patterns
- MIME header decoding supports international characters (Base64, Quoted-Printable)
- Storage backend automatically applies gzip compression to text files (.eml, .json) for significant space savings in S3 buckets, while leaving binary files uncompressed
- Webhook payloads include both original email data and generated markdown content