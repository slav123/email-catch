#!/bin/bash

# Test script for HIB.PL email capture deployment
# Usage: ./scripts/test-hib-deployment.sh

SERVER="capture.hib.pl"
TEST_EMAIL="test@capture.hib.pl"

echo "🧪 Testing Email Catch deployment on $SERVER"
echo "================================================"

# Test 1: Check if server is responding
echo "1️⃣ Testing server connectivity..."
if curl -s --connect-timeout 5 http://$SERVER > /dev/null; then
    echo "✅ Server is accessible"
else
    echo "❌ Server is not accessible"
    exit 1
fi

# Test 2: Check SMTP ports
echo ""
echo "2️⃣ Testing SMTP ports..."
for port in 25 587 465; do
    echo -n "   Port $port: "
    if timeout 5 bash -c "</dev/tcp/$SERVER/$port" 2>/dev/null; then
        echo "✅ Open"
    else
        echo "❌ Closed or filtered"
    fi
done

# Test 3: Check HTTP port (for Let's Encrypt)
echo ""
echo "3️⃣ Testing HTTP port for Let's Encrypt..."
if timeout 5 bash -c "</dev/tcp/$SERVER/80" 2>/dev/null; then
    echo "✅ Port 80 is open"
else
    echo "❌ Port 80 is closed (needed for Let's Encrypt)"
fi

# Test 4: DNS resolution
echo ""
echo "4️⃣ Testing DNS resolution..."
if dig +short $SERVER > /dev/null; then
    IP=$(dig +short $SERVER | head -1)
    echo "✅ DNS resolves to: $IP"
else
    echo "❌ DNS resolution failed"
fi

# Test 5: MX record check
echo ""
echo "5️⃣ Testing MX records..."
if dig +short MX hib.pl | grep -q "capture.hib.pl"; then
    echo "✅ MX record for hib.pl points to capture.hib.pl"
else
    echo "⚠️  MX record for hib.pl should point to capture.hib.pl"
fi

if dig +short MX capture.hib.pl | grep -q "capture.hib.pl"; then
    echo "✅ MX record for capture.hib.pl points to capture.hib.pl"
else
    echo "⚠️  MX record for capture.hib.pl should point to capture.hib.pl"
fi

# Test 6: SSL Certificate check
echo ""
echo "6️⃣ Testing SSL certificate..."
if openssl s_client -connect $SERVER:465 -servername $SERVER -verify_return_error </dev/null 2>/dev/null; then
    echo "✅ SSL certificate is valid"
    
    # Get certificate details
    CERT_INFO=$(openssl s_client -connect $SERVER:465 -servername $SERVER </dev/null 2>/dev/null | openssl x509 -noout -dates 2>/dev/null)
    echo "   $CERT_INFO"
else
    echo "⚠️  SSL certificate issue (might be Let's Encrypt staging or not yet configured)"
fi

# Test 7: SMTP conversation
echo ""
echo "7️⃣ Testing SMTP conversation..."
SMTP_TEST=$(timeout 10 bash -c "
echo 'EHLO test.example.com
QUIT' | nc $SERVER 25 2>/dev/null
")

if echo "$SMTP_TEST" | grep -q "220.*ESMTP"; then
    echo "✅ SMTP server responds correctly"
    echo "   Response: $(echo "$SMTP_TEST" | head -1)"
else
    echo "❌ SMTP server not responding correctly"
    echo "   Response: $SMTP_TEST"
fi

# Test 8: Send test email (if server is responding)
echo ""
echo "8️⃣ Sending test email..."
if command -v sendmail >/dev/null 2>&1; then
    echo "Subject: Test Email from Deployment Test
From: test@example.com
To: $TEST_EMAIL

This is a test email sent during deployment verification.
Timestamp: $(date)
Test ID: test-$(date +%s)" | sendmail $TEST_EMAIL

    echo "✅ Test email sent to $TEST_EMAIL"
    echo "   Check server logs and storage to verify delivery"
else
    echo "⚠️  sendmail not available, skipping email test"
    echo "   You can test manually with: echo 'Test' | sendmail $TEST_EMAIL"
fi

# Test 9: Check for common issues
echo ""
echo "9️⃣ Checking for common issues..."

# Check if port 25 might be blocked by ISP
echo -n "   ISP port 25 blocking check: "
if timeout 5 bash -c "</dev/tcp/smtp.gmail.com/25" 2>/dev/null; then
    echo "✅ Port 25 outbound works (not blocked by ISP)"
else
    echo "⚠️  Port 25 might be blocked by ISP"
fi

# Summary
echo ""
echo "📋 Test Summary"
echo "==============="
echo "Server: $SERVER"
echo "Test email: $TEST_EMAIL"
echo ""
echo "Next steps:"
echo "1. Check server logs: ssh root@$SERVER 'journalctl -u email-catch -f'"
echo "2. Check certificate: ssh root@$SERVER '/opt/email-catch/bin/certmanager -action info'"
echo "3. Monitor email storage in Cloudflare R2"
echo "4. Test with real email clients"
echo ""
echo "🎉 Deployment test completed!"