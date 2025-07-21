# Email Catch Deployment Guide for capture.hib.pl

This guide covers deploying Email Catch on the `capture.hib.pl` server with Cloudflare R2 storage.

## Prerequisites

1. **Server Requirements**
   - Server with public IP for `capture.hib.pl`
   - Root access to the server
   - Ports 25, 80, 465, 587 open in firewall

2. **DNS Configuration**
   - A record for `capture.hib.pl` pointing to your server IP
   - MX record for `hib.pl` pointing to `capture.hib.pl`
   - MX record for `capture.hib.pl` pointing to `capture.hib.pl`

## DNS Setup

Add these DNS records to your domain:

```dns
# A Record
capture.hib.pl.    IN  A     YOUR_SERVER_IP

# MX Records
hib.pl.            IN  MX    10  capture.hib.pl.
capture.hib.pl.    IN  MX    10  capture.hib.pl.

# Optional: SPF Record
hib.pl.            IN  TXT   "v=spf1 mx ~all"
capture.hib.pl.    IN  TXT   "v=spf1 mx ~all"
```

## Installation Steps

### 1. Prepare Server

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y curl wget unzip

# Create dedicated user
sudo useradd -r -s /bin/false email-catch

# Create application directories
sudo mkdir -p /opt/email-catch/{bin,config,emails,test-emails,logs,certs}
sudo chown -R email-catch:email-catch /opt/email-catch
```

### 2. Upload and Install Application

```bash
# Copy built binaries to server
scp bin/email-catch root@capture.hib.pl:/opt/email-catch/bin/
scp bin/certmanager root@capture.hib.pl:/opt/email-catch/bin/

# Copy configuration files
scp config/config-production-hib.yaml root@capture.hib.pl:/opt/email-catch/config/
scp config/config-staging-hib.yaml root@capture.hib.pl:/opt/email-catch/config/

# Copy systemd service
scp deploy/systemd-hib.service root@capture.hib.pl:/etc/systemd/system/email-catch.service

# Set permissions
sudo chmod +x /opt/email-catch/bin/*
sudo chown -R email-catch:email-catch /opt/email-catch
```

### 3. Configure Email Address

Edit the configuration file and update the email address:

```bash
sudo nano /opt/email-catch/config/config-production-hib.yaml
```

Change this line to your actual email:
```yaml
email: "admin@hib.pl"  # Update to your real email address
```

### 4. Test with Staging Environment First

```bash
# Start with staging configuration to test Let's Encrypt
sudo systemctl daemon-reload
sudo systemctl enable email-catch

# Temporarily use staging config for testing
sudo sed -i 's/config-production-hib.yaml/config-staging-hib.yaml/' /etc/systemd/system/email-catch.service
sudo systemctl daemon-reload

# Start service
sudo systemctl start email-catch

# Check status
sudo systemctl status email-catch

# Check logs
sudo journalctl -u email-catch -f
```

### 5. Test Certificate Generation

```bash
# Test certificate generation with staging
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-staging-hib.yaml -action renew

# Check certificate info
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-staging-hib.yaml -action info
```

### 6. Switch to Production

Once staging works correctly:

```bash
# Stop service
sudo systemctl stop email-catch

# Switch back to production config
sudo sed -i 's/config-staging-hib.yaml/config-production-hib.yaml/' /etc/systemd/system/email-catch.service
sudo systemctl daemon-reload

# Clean staging certificates
sudo rm -rf /opt/email-catch/certs/letsencrypt/*

# Start production service
sudo systemctl start email-catch

# Check status
sudo systemctl status email-catch
```

### 7. Configure Firewall

```bash
# UFW example
sudo ufw allow 25/tcp    # SMTP
sudo ufw allow 80/tcp    # HTTP (for Let's Encrypt)
sudo ufw allow 465/tcp   # SMTPS
sudo ufw allow 587/tcp   # SMTP submission
sudo ufw allow 2525/tcp  # Alternative SMTP

# Or iptables
sudo iptables -A INPUT -p tcp --dport 25 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 465 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 587 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 2525 -j ACCEPT
```

## Testing the Deployment

### 1. Test SMTP Connectivity

```bash
# Test SMTP port 25
telnet capture.hib.pl 25

# Expected response:
# 220 capture.hib.pl ESMTP Ready

# Test SMTPS port 465
openssl s_client -connect capture.hib.pl:465 -servername capture.hib.pl
```

### 2. Send Test Emails

```bash
# Send test email via command line
echo "Subject: Test Email\n\nThis is a test email." | sendmail test@capture.hib.pl

# Or use the test client
/opt/email-catch/bin/test-email-sender -host capture.hib.pl -ports "25" -type simple -count 1 -to "test@capture.hib.pl"
```

### 3. Verify Email Storage

```bash
# Check local storage
sudo ls -la /opt/email-catch/emails/

# Check Cloudflare R2 via logs
sudo journalctl -u email-catch | grep -i "s3\|cloudflare\|upload"
```

### 4. Test Different Email Patterns

```bash
# Test hib.pl domain
echo "Test" | sendmail admin@hib.pl

# Test capture subdomain
echo "Test" | sendmail user@capture.hib.pl

# Test support emails
echo "Test" | sendmail support@hib.pl
```

## Monitoring and Maintenance

### 1. View Logs

```bash
# Real-time logs
sudo journalctl -u email-catch -f

# Error logs only
sudo journalctl -u email-catch | grep -i error

# Certificate-related logs
sudo journalctl -u email-catch | grep -i certificate
```

### 2. Check Certificate Status

```bash
# View certificate information
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-production-hib.yaml -action info

# Manual renewal if needed
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-production-hib.yaml -action renew
```

### 3. Monitor Cloudflare R2 Storage

```bash
# Check for S3 upload errors in logs
sudo journalctl -u email-catch | grep -i "s3.*error\|upload.*failed"

# Check email processing statistics
sudo journalctl -u email-catch | grep "Processing email"
```

### 4. Log Rotation

Create `/etc/logrotate.d/email-catch`:

```
/opt/email-catch/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 email-catch email-catch
    postrotate
        systemctl reload email-catch
    endscript
}
```

## Configuration Details

### Cloudflare R2 Storage Structure

Your emails will be stored in Cloudflare R2 with this structure:

```
email/
├── emails/
│   ├── hib-domain/          # Emails to @hib.pl
│   ├── capture/             # Emails to @capture.hib.pl
│   ├── support/             # Support emails
│   ├── testing/             # Test emails
│   └── general/             # All other emails
└── test-emails/             # Staging/test emails
```

### Email Routing Rules

The configuration includes these routing rules:

1. **HIB Domain**: `.*@hib\.pl` → `hib-domain/` folder
2. **Capture Subdomain**: `.*@capture\.hib\.pl` → `capture/` folder
3. **Support Emails**: `(support|admin|postmaster)@.*` → `support/` folder
4. **Test Emails**: `test@.*` → `testing/` folder
5. **Catch-all**: `.*` → `general/` folder

## Troubleshooting

### Common Issues

1. **Let's Encrypt Challenge Failed**
   ```bash
   # Check if port 80 is accessible
   curl -I http://capture.hib.pl/.well-known/acme-challenge/test
   
   # Verify DNS
   dig capture.hib.pl
   ```

2. **SMTP Port Access Denied**
   ```bash
   # Check if service has permission to bind to port 25
   sudo setcap 'cap_net_bind_service=+ep' /opt/email-catch/bin/email-catch
   ```

3. **Cloudflare R2 Connection Issues**
   ```bash
   # Test R2 connectivity
   sudo journalctl -u email-catch | grep -i "s3\|cloudflare"
   ```

### Getting Help

1. Check service status: `sudo systemctl status email-catch`
2. View logs: `sudo journalctl -u email-catch -f`
3. Test configuration: `sudo -u email-catch /opt/email-catch/bin/email-catch -config /opt/email-catch/config/config-production-hib.yaml --help`

## Security Notes

- Certificates are automatically renewed by Let's Encrypt
- All email data is stored encrypted in Cloudflare R2
- Local backups are maintained in `/opt/email-catch/emails`
- Service runs as non-root user with minimal privileges
- Rate limiting is enabled to prevent abuse

Your email capture service for `capture.hib.pl` is now ready for production use!