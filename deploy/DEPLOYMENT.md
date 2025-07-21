# Email Catch Production Deployment Guide

This guide covers deploying Email Catch in a production environment with Let's Encrypt SSL certificates.

## Prerequisites

1. **Server Requirements**
   - Ubuntu/Debian/CentOS server with root access
   - Public IP address
   - Domain name pointing to your server
   - Ports 25, 80, 465, 587 open in firewall

2. **DNS Configuration**
   - MX record pointing to your server
   - A record for mail subdomain (e.g., `mail.yourdomain.com`)

## Installation Steps

### 1. Create User and Directories

```bash
# Create dedicated user
sudo useradd -r -s /bin/false email-catch

# Create application directory
sudo mkdir -p /opt/email-catch/{bin,config,emails,logs,certs}
sudo chown -R email-catch:email-catch /opt/email-catch
```

### 2. Install Application

```bash
# Copy built binaries
sudo cp bin/email-catch /opt/email-catch/bin/
sudo cp bin/certmanager /opt/email-catch/bin/
sudo chmod +x /opt/email-catch/bin/*

# Copy configuration
sudo cp config/config-letsencrypt.yaml /opt/email-catch/config/
sudo chown email-catch:email-catch /opt/email-catch/config/*
```

### 3. Configure Let's Encrypt

Edit `/opt/email-catch/config/config-letsencrypt.yaml`:

```yaml
server:
  hostname: "mail.yourdomain.com"  # Your actual domain
  tls:
    letsencrypt:
      enabled: true
      domains: ["mail.yourdomain.com"]
      email: "admin@yourdomain.com"  # Your email for Let's Encrypt
      staging: false  # Set to true for testing first
```

### 4. Test Certificate Generation

```bash
# Test with Let's Encrypt staging first
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-letsencrypt.yaml -action renew

# Check certificate info
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-letsencrypt.yaml -action info
```

### 5. Install Systemd Service

```bash
# Copy service file
sudo cp deploy/email-catch.service /etc/systemd/system/

# Reload systemd and enable service
sudo systemctl daemon-reload
sudo systemctl enable email-catch
```

### 6. Configure Firewall

```bash
# UFW example
sudo ufw allow 25/tcp    # SMTP
sudo ufw allow 80/tcp    # HTTP (for Let's Encrypt)
sudo ufw allow 465/tcp   # SMTPS
sudo ufw allow 587/tcp   # SMTP submission
sudo ufw allow 2525/tcp  # Alternative SMTP (optional)

# iptables example
sudo iptables -A INPUT -p tcp --dport 25 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 465 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 587 -j ACCEPT
```

### 7. Start Service

```bash
# Start the service
sudo systemctl start email-catch

# Check status
sudo systemctl status email-catch

# View logs
sudo journalctl -u email-catch -f
```

## Post-Deployment

### 1. Verify Installation

```bash
# Test SMTP connection
telnet mail.yourdomain.com 25

# Check certificate
openssl s_client -connect mail.yourdomain.com:465 -servername mail.yourdomain.com

# Send test email
echo "Subject: Test Email\n\nThis is a test." | sendmail test@yourdomain.com
```

### 2. Configure DNS Records

```dns
# MX Record
yourdomain.com.    IN  MX  10  mail.yourdomain.com.

# A Record
mail.yourdomain.com.  IN  A   YOUR_SERVER_IP

# Optional: SPF Record
yourdomain.com.    IN  TXT  "v=spf1 mx ~all"
```

### 3. Monitor Certificate Renewal

Let's Encrypt certificates are automatically renewed. Monitor the logs:

```bash
# Check renewal logs
sudo journalctl -u email-catch | grep -i certificate

# Manual renewal test
sudo -u email-catch /opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-letsencrypt.yaml -action renew
```

## Configuration Examples

### Production Configuration

```yaml
server:
  ports: [25, 587, 465]
  hostname: "mail.yourdomain.com"
  tls:
    enabled: true
    letsencrypt:
      enabled: true
      domains: ["mail.yourdomain.com"]
      email: "admin@yourdomain.com"
      staging: false
      http_port: 80
      renew_before_days: 30
  rate_limit:
    enabled: true
    max_emails_per_minute: 1000
    max_email_size_mb: 50

storage:
  s3_compatible:
    enabled: true
    endpoint: "s3.amazonaws.com"
    access_key: "${S3_ACCESS_KEY}"
    secret_key: "${S3_SECRET_KEY}"
    bucket: "email-catch-prod"
    region: "us-east-1"
    use_ssl: true
```

### Environment Variables

Create `/opt/email-catch/.env`:

```bash
S3_ACCESS_KEY=your_s3_access_key
S3_SECRET_KEY=your_s3_secret_key
WEBHOOK_TOKEN=your_webhook_token
```

Load in systemd service:

```ini
[Service]
EnvironmentFile=/opt/email-catch/.env
```

## Monitoring and Maintenance

### Log Rotation

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

### Health Check Script

```bash
#!/bin/bash
# /opt/email-catch/scripts/health-check.sh

# Check if service is running
if ! systemctl is-active --quiet email-catch; then
    echo "Email Catch service is not running"
    exit 1
fi

# Check if ports are listening
for port in 25 587 465; do
    if ! netstat -tuln | grep ":$port " > /dev/null; then
        echo "Port $port is not listening"
        exit 1
    fi
done

# Check certificate expiry
cert_info=$(/opt/email-catch/bin/certmanager -config /opt/email-catch/config/config-letsencrypt.yaml -action info 2>/dev/null)
if echo "$cert_info" | grep -q "error"; then
    echo "Certificate error detected"
    exit 1
fi

echo "All checks passed"
exit 0
```

### Backup Strategy

```bash
#!/bin/bash
# /opt/email-catch/scripts/backup.sh

# Backup configuration
tar -czf /backup/email-catch-config-$(date +%Y%m%d).tar.gz -C /opt/email-catch config

# Backup certificates
tar -czf /backup/email-catch-certs-$(date +%Y%m%d).tar.gz -C /opt/email-catch certs

# Backup recent emails (last 7 days)
find /opt/email-catch/emails -name "*.eml" -mtime -7 | tar -czf /backup/email-catch-emails-$(date +%Y%m%d).tar.gz -T -
```

## Troubleshooting

### Common Issues

1. **Port 25 Access Denied**
   ```bash
   # Check if running as root or with CAP_NET_BIND_SERVICE
   sudo setcap 'cap_net_bind_service=+ep' /opt/email-catch/bin/email-catch
   ```

2. **Let's Encrypt Challenge Failed**
   ```bash
   # Ensure port 80 is accessible
   curl -I http://mail.yourdomain.com/.well-known/acme-challenge/test
   
   # Check DNS
   dig mail.yourdomain.com
   ```

3. **Certificate Not Renewing**
   ```bash
   # Check renewal logs
   sudo journalctl -u email-catch | grep -i renew
   
   # Manual renewal
   sudo systemctl stop email-catch
   sudo -u email-catch /opt/email-catch/bin/certmanager -action renew
   sudo systemctl start email-catch
   ```

### Log Analysis

```bash
# View real-time logs
sudo journalctl -u email-catch -f

# Filter for errors
sudo journalctl -u email-catch | grep -i error

# View certificate-related logs
sudo journalctl -u email-catch | grep -i certificate
```

## Security Considerations

1. **Firewall**: Only open necessary ports
2. **User Permissions**: Run as non-root user
3. **File Permissions**: Restrict access to configuration and certificates
4. **Rate Limiting**: Configure appropriate limits
5. **Monitoring**: Set up alerts for service failures
6. **Updates**: Keep the application and system updated

## Performance Tuning

1. **File Descriptors**: Increase limits in systemd service
2. **Email Storage**: Use S3 for production to avoid disk space issues
3. **Log Rotation**: Prevent log files from consuming disk space
4. **Resource Limits**: Set appropriate memory and CPU limits

For support and updates, see: https://github.com/slav123/email-catch