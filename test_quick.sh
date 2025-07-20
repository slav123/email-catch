#!/bin/bash

set -e

echo "🚀 Starting Email Catch Quick Test"

# Build the application
echo "📦 Building email-catch..."
go build -o bin/email-catch ./cmd/server

# Build test tools
echo "🔧 Building test tools..."
go build -o bin/test-email-sender ./tests/scripts/test_email_sender.go

# Create necessary directories
mkdir -p emails logs test-emails

# Start the server in background with test config
echo "🌟 Starting server with test configuration..."
./bin/email-catch -config config/test-config.yaml &
SERVER_PID=$!

# Give server time to start
sleep 2

# Send a simple test email
echo "📧 Sending test email..."
./bin/test-email-sender -ports "2525" -type simple -count 1 -to "test@test.com" -from "sender@example.com"

# Wait a moment for processing
sleep 1

# Check if email was saved
echo "🔍 Checking for saved email..."
if [ -n "$(find test-emails -name '*.eml' 2>/dev/null)" ]; then
    echo "✅ SUCCESS: Email was captured and saved!"
    echo "📁 Saved files:"
    find test-emails -name '*.eml' -exec ls -la {} \;
else
    echo "❌ FAIL: No email files found"
fi

# Stop the server
echo "🛑 Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "🎉 Quick test completed!"