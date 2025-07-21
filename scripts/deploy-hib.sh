#!/bin/bash

# Email Catch Deployment Script for capture.hib.pl
# Usage: ./scripts/deploy-hib.sh [staging|production]

set -e

MODE=${1:-staging}
SERVER="capture.hib.pl"
REMOTE_USER="root"
LOCAL_PATH="/Users/slav/go/src/github.com/slav123/email-catch"

echo "🚀 Deploying Email Catch to $SERVER in $MODE mode..."

# Build the application locally
echo "📦 Building application..."
go build -o bin/email-catch ./cmd/server
go build -o bin/certmanager ./cmd/certmanager

# Create deployment package
echo "📋 Creating deployment package..."
mkdir -p deploy/package
cp bin/email-catch deploy/package/
cp bin/certmanager deploy/package/
cp config/config-*-hib.yaml deploy/package/
cp deploy/systemd-hib.service deploy/package/

# Upload to server
echo "📤 Uploading to server..."
scp -r deploy/package/* $REMOTE_USER@$SERVER:/tmp/email-catch-deploy/

# Execute deployment on server
echo "🔧 Executing deployment on server..."
ssh $REMOTE_USER@$SERVER << 'ENDSSH'
set -e

echo "Creating user and directories..."
if ! id "email-catch" &>/dev/null; then
    useradd -r -s /bin/false email-catch
fi

mkdir -p /opt/email-catch/{bin,config,emails,test-emails,logs,certs}

echo "Installing binaries..."
cp /tmp/email-catch-deploy/email-catch /opt/email-catch/bin/
cp /tmp/email-catch-deploy/certmanager /opt/email-catch/bin/
chmod +x /opt/email-catch/bin/*

echo "Installing configuration..."
cp /tmp/email-catch-deploy/config-*-hib.yaml /opt/email-catch/config/

echo "Installing systemd service..."
cp /tmp/email-catch-deploy/systemd-hib.service /etc/systemd/system/email-catch.service

echo "Setting permissions..."
chown -R email-catch:email-catch /opt/email-catch

echo "Configuring systemd..."
systemctl daemon-reload
systemctl enable email-catch

echo "Cleaning up..."
rm -rf /tmp/email-catch-deploy

echo "✅ Deployment completed!"
ENDSSH

# Configure for staging or production mode
if [ "$MODE" = "staging" ]; then
    echo "🧪 Configuring for STAGING mode..."
    ssh $REMOTE_USER@$SERVER << 'ENDSSH'
sed -i 's/config-production-hib.yaml/config-staging-hib.yaml/' /etc/systemd/system/email-catch.service
systemctl daemon-reload
ENDSSH
elif [ "$MODE" = "production" ]; then
    echo "🏭 Configuring for PRODUCTION mode..."
    ssh $REMOTE_USER@$SERVER << 'ENDSSH'
sed -i 's/config-staging-hib.yaml/config-production-hib.yaml/' /etc/systemd/system/email-catch.service
systemctl daemon-reload
ENDSSH
fi

# Start the service
echo "🎬 Starting email-catch service..."
ssh $REMOTE_USER@$SERVER << 'ENDSSH'
systemctl restart email-catch
sleep 2
systemctl status email-catch
ENDSSH

echo ""
echo "🎉 Deployment completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Check service status: ssh $REMOTE_USER@$SERVER 'systemctl status email-catch'"
echo "2. View logs: ssh $REMOTE_USER@$SERVER 'journalctl -u email-catch -f'"
echo "3. Test SMTP: telnet $SERVER 25"
echo "4. Check certificate: ssh $REMOTE_USER@$SERVER '/opt/email-catch/bin/certmanager -action info'"
echo ""
echo "📧 Test email:"
echo "echo 'Subject: Test Email\n\nTest message' | sendmail test@capture.hib.pl"
echo ""

# Clean up local deployment package
rm -rf deploy/package

echo "🏁 Deployment script finished!"