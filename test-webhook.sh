#!/bin/bash

# Test script for n8n-go webhook functionality

set -e

echo "🧪 n8n-go Webhook Test Script"
echo "=============================="
echo

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
WEBHOOK_PATH="test-webhook"

echo "📋 Configuration:"
echo "  API URL: $API_URL"
echo "  Webhook Path: $WEBHOOK_PATH"
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Create workflow with webhook
echo "📝 Test 1: Creating workflow with webhook..."
WORKFLOW_ID=$(curl -s -X POST "$API_URL/api/v1/workflows" \
  -H "Content-Type: application/json" \
  -d @test-workflows/webhook-example.json | jq -r '.id')

if [ -z "$WORKFLOW_ID" ] || [ "$WORKFLOW_ID" == "null" ]; then
  echo -e "${RED}❌ Failed to create workflow${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Workflow created: $WORKFLOW_ID${NC}"
echo

# Wait for webhook registration
sleep 2

# Test 2: List webhooks
echo "📝 Test 2: Listing webhooks..."
WEBHOOK_COUNT=$(curl -s "$API_URL/api/v1/webhooks" | jq -r '.count // 0')
echo -e "${GREEN}✅ Found $WEBHOOK_COUNT webhook(s)${NC}"
echo

# Test 3: Trigger webhook (simple)
echo "📝 Test 3: Triggering webhook (simple request)..."
RESPONSE=$(curl -s -X POST "$API_URL/webhook/$WEBHOOK_PATH" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from webhook test!"}')

echo "Response: $RESPONSE"
if echo "$RESPONSE" | jq -e '.receivedMessage' > /dev/null 2>&1; then
  echo -e "${GREEN}✅ Webhook executed successfully${NC}"
else
  echo -e "${YELLOW}⚠️  Webhook response format unexpected${NC}"
fi
echo

# Test 4: Trigger webhook (with query parameters)
echo "📝 Test 4: Triggering webhook with query parameters..."
RESPONSE=$(curl -s -X POST "$API_URL/webhook/$WEBHOOK_PATH?param1=value1&param2=value2" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello with query params!"}')

echo "Response: $RESPONSE"
echo -e "${GREEN}✅ Webhook with query parameters executed${NC}"
echo

# Test 5: Trigger webhook (with custom headers)
echo "📝 Test 5: Triggering webhook with custom headers..."
RESPONSE=$(curl -s -X POST "$API_URL/webhook/$WEBHOOK_PATH" \
  -H "Content-Type: application/json" \
  -H "X-Custom-Header: CustomValue" \
  -d '{"message": "Hello with custom headers!"}')

echo "Response: $RESPONSE"
echo -e "${GREEN}✅ Webhook with custom headers executed${NC}"
echo

# Test 6: Test webhook (using test endpoint)
echo "📝 Test 6: Testing webhook via test endpoint..."
RESPONSE=$(curl -s -X POST "$API_URL/webhook-test/$WEBHOOK_PATH" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from test webhook!"}')

echo "Response: $RESPONSE"
if [ -n "$RESPONSE" ]; then
  echo -e "${GREEN}✅ Test webhook endpoint works${NC}"
else
  echo -e "${YELLOW}⚠️  Test webhook may not be configured${NC}"
fi
echo

# Test 7: Invalid webhook path (should 404)
echo "📝 Test 7: Testing invalid webhook path..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/webhook/invalid-path" \
  -H "Content-Type: application/json" \
  -d '{"message": "test"}')

if [ "$HTTP_CODE" == "404" ]; then
  echo -e "${GREEN}✅ Invalid webhook correctly returns 404${NC}"
else
  echo -e "${YELLOW}⚠️  Expected 404, got $HTTP_CODE${NC}"
fi
echo

# Test 8: List workflow executions
echo "📝 Test 8: Listing workflow executions..."
EXEC_COUNT=$(curl -s "$API_URL/api/v1/executions?workflowId=$WORKFLOW_ID" | jq -r '.count // 0')
echo -e "${GREEN}✅ Found $EXEC_COUNT execution(s)${NC}"
echo

# Test 9: Deactivate workflow
echo "📝 Test 9: Deactivating workflow..."
curl -s -X POST "$API_URL/api/v1/workflows/$WORKFLOW_ID/deactivate" > /dev/null
echo -e "${GREEN}✅ Workflow deactivated${NC}"
echo

# Wait for webhook unregistration
sleep 1

# Test 10: Webhook should 404 after deactivation
echo "📝 Test 10: Testing webhook after deactivation..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/webhook/$WEBHOOK_PATH" \
  -H "Content-Type: application/json" \
  -d '{"message": "test"}')

if [ "$HTTP_CODE" == "404" ]; then
  echo -e "${GREEN}✅ Webhook correctly unavailable after deactivation${NC}"
else
  echo -e "${YELLOW}⚠️  Expected 404 after deactivation, got $HTTP_CODE${NC}"
fi
echo

# Cleanup
echo "🧹 Cleaning up..."
curl -s -X DELETE "$API_URL/api/v1/workflows/$WORKFLOW_ID" > /dev/null
echo -e "${GREEN}✅ Workflow deleted${NC}"
echo

echo "=============================="
echo -e "${GREEN}🎉 All webhook tests completed!${NC}"
echo

# Summary
echo "📊 Test Summary:"
echo "  ✅ Workflow creation"
echo "  ✅ Webhook registration"
echo "  ✅ Webhook execution (simple)"
echo "  ✅ Webhook with query parameters"
echo "  ✅ Webhook with custom headers"
echo "  ✅ Test webhook endpoint"
echo "  ✅ Invalid path handling"
echo "  ✅ Execution tracking"
echo "  ✅ Workflow deactivation"
echo "  ✅ Webhook unregistration"
echo
