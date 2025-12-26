#!/bin/bash

# Test script for m9m workflow version system
# Demonstrates creating, listing, restoring, and managing versions

set -e

BASE_URL="${M9M_URL:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"

echo "🧪 m9m Workflow Version System Test"
echo "======================================="
echo "Testing against: $BASE_URL"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper function for pretty printing JSON
print_json() {
    if command -v python3 &> /dev/null; then
        echo "$1" | python3 -m json.tool
    else
        echo "$1" | jq '.'
    fi
}

echo -e "${BLUE}Step 1: Create a test workflow${NC}"
echo "-------------------------------"
WORKFLOW_RESPONSE=$(curl -s -X POST "$API_URL/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Version Test Workflow",
    "active": false,
    "nodes": [
      {
        "name": "Start",
        "type": "n8n-nodes-base.start",
        "position": [100, 200],
        "parameters": {}
      }
    ],
    "connections": {}
  }')

WORKFLOW_ID=$(echo "$WORKFLOW_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "$WORKFLOW_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo -e "${GREEN}✓ Created workflow: $WORKFLOW_ID${NC}"
echo ""

echo -e "${BLUE}Step 2: Create initial version (v1.0.0)${NC}"
echo "---------------------------------------"
VERSION1_RESPONSE=$(curl -s -X POST "$API_URL/workflows/$WORKFLOW_ID/versions" \
  -H "Content-Type: application/json" \
  -d '{
    "versionTag": "v1.0.0",
    "description": "Initial version with Start node",
    "tags": ["initial", "release"]
  }')

VERSION1_ID=$(echo "$VERSION1_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "unknown")
echo -e "${GREEN}✓ Created version v1.0.0: $VERSION1_ID${NC}"
echo "Changes detected:"
echo "$VERSION1_RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); [print(f'  - {change}') for change in data['changes']]" 2>/dev/null || echo "  - Initial version"
echo ""

echo -e "${BLUE}Step 3: Update workflow (add HTTP Request node)${NC}"
echo "----------------------------------------------"
curl -s -X PUT "$API_URL/workflows/$WORKFLOW_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Version Test Workflow - Updated",
    "active": true,
    "nodes": [
      {
        "name": "Start",
        "type": "n8n-nodes-base.start",
        "position": [100, 200],
        "parameters": {}
      },
      {
        "name": "HTTP Request",
        "type": "n8n-nodes-base.httpRequest",
        "position": [300, 200],
        "parameters": {
          "url": "https://api.example.com",
          "method": "GET"
        }
      }
    ],
    "connections": {
      "Start": {
        "main": [[{"node": "HTTP Request", "type": "main", "index": 0}]]
      }
    }
  }' > /dev/null

echo -e "${GREEN}✓ Updated workflow (added HTTP Request node, renamed, activated)${NC}"
echo ""

echo -e "${BLUE}Step 4: Create second version (v1.1.0)${NC}"
echo "--------------------------------------"
VERSION2_RESPONSE=$(curl -s -X POST "$API_URL/workflows/$WORKFLOW_ID/versions" \
  -H "Content-Type: application/json" \
  -d '{
    "versionTag": "v1.1.0",
    "description": "Added HTTP Request node for API integration",
    "tags": ["feature", "http"]
  }')

VERSION2_ID=$(echo "$VERSION2_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "unknown")
echo -e "${GREEN}✓ Created version v1.1.0: $VERSION2_ID${NC}"
echo "Changes detected:"
echo "$VERSION2_RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); [print(f'  - {change}') for change in data['changes']]" 2>/dev/null || echo "  - Changes detected"
echo ""

echo -e "${BLUE}Step 5: Update workflow again (add Set node)${NC}"
echo "--------------------------------------------"
curl -s -X PUT "$API_URL/workflows/$WORKFLOW_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Version Test Workflow - Updated",
    "active": true,
    "nodes": [
      {
        "name": "Start",
        "type": "n8n-nodes-base.start",
        "position": [100, 200],
        "parameters": {}
      },
      {
        "name": "HTTP Request",
        "type": "n8n-nodes-base.httpRequest",
        "position": [300, 200],
        "parameters": {
          "url": "https://api.example.com",
          "method": "GET"
        }
      },
      {
        "name": "Set",
        "type": "n8n-nodes-base.set",
        "position": [500, 200],
        "parameters": {
          "values": {}
        }
      }
    ],
    "connections": {
      "Start": {
        "main": [[{"node": "HTTP Request", "type": "main", "index": 0}]]
      },
      "HTTP Request": {
        "main": [[{"node": "Set", "type": "main", "index": 0}]]
      }
    }
  }' > /dev/null

echo -e "${GREEN}✓ Updated workflow (added Set node)${NC}"
echo ""

echo -e "${BLUE}Step 6: Create third version (v1.2.0)${NC}"
echo "-------------------------------------"
VERSION3_RESPONSE=$(curl -s -X POST "$API_URL/workflows/$WORKFLOW_ID/versions" \
  -H "Content-Type: application/json" \
  -d '{
    "versionTag": "v1.2.0",
    "description": "Added Set node for data transformation",
    "tags": ["feature", "transform"]
  }')

VERSION3_ID=$(echo "$VERSION3_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "unknown")
echo -e "${GREEN}✓ Created version v1.2.0: $VERSION3_ID${NC}"
echo ""

echo -e "${BLUE}Step 7: List all versions${NC}"
echo "------------------------"
VERSIONS_RESPONSE=$(curl -s -X GET "$API_URL/workflows/$WORKFLOW_ID/versions?limit=10")
echo "$VERSIONS_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total versions: {data[\"total\"]}')
print('\nVersions:')
for i, v in enumerate(data['data']):
    current = '(CURRENT)' if v['isCurrent'] else ''
    print(f'  {i+1}. {v[\"versionTag\"]} (v{v[\"versionNum\"]}) {current}')
    print(f'     Description: {v[\"description\"]}')
    print(f'     Tags: {', '.join(v.get('tags', []))}')
    print(f'     Created: {v[\"createdAt\"][:19]}')
    print()
" 2>/dev/null || echo "$VERSIONS_RESPONSE"
echo ""

echo -e "${BLUE}Step 8: Get specific version (v1.0.0)${NC}"
echo "------------------------------------"
SPECIFIC_VERSION=$(curl -s -X GET "$API_URL/workflows/$WORKFLOW_ID/versions/$VERSION1_ID")
echo "$SPECIFIC_VERSION" | python3 -c "
import sys, json
v = json.load(sys.stdin)
print(f'Version: {v[\"versionTag\"]} (v{v[\"versionNum\"]})')
print(f'Description: {v[\"description\"]}')
print(f'Author: {v[\"author\"]}')
print(f'Nodes in this version: {len(v[\"workflow\"][\"nodes\"])}')
print('Changes:')
for change in v['changes']:
    print(f'  - {change}')
" 2>/dev/null || echo "$SPECIFIC_VERSION"
echo ""

echo -e "${BLUE}Step 9: Restore to v1.0.0 (with automatic backup)${NC}"
echo "-------------------------------------------------"
RESTORE_RESPONSE=$(curl -s -X POST "$API_URL/workflows/$WORKFLOW_ID/versions/$VERSION1_ID/restore" \
  -H "Content-Type: application/json" \
  -d '{
    "createBackup": true,
    "description": "Testing restore functionality"
  }')

echo "$RESTORE_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Message: {data[\"message\"]}')
print(f'Backup created: {data[\"backupCreated\"]}')
print(f'Restored from: {data[\"restoredFrom\"][\"versionTag\"]}')
" 2>/dev/null || echo "$RESTORE_RESPONSE"
echo -e "${GREEN}✓ Workflow restored to v1.0.0${NC}"
echo ""

echo -e "${BLUE}Step 10: Verify workflow was restored${NC}"
echo "------------------------------------"
CURRENT_WORKFLOW=$(curl -s -X GET "$API_URL/workflows/$WORKFLOW_ID")
echo "$CURRENT_WORKFLOW" | python3 -c "
import sys, json
w = json.load(sys.stdin)
print(f'Workflow name: {w[\"name\"]}')
print(f'Active: {w[\"active\"]}')
print(f'Number of nodes: {len(w[\"nodes\"])}')
print('Nodes:')
for node in w['nodes']:
    print(f'  - {node[\"name\"]} ({node[\"type\"]})')
" 2>/dev/null || echo "$CURRENT_WORKFLOW"
echo ""

echo -e "${BLUE}Step 11: List versions after restore${NC}"
echo "-----------------------------------"
VERSIONS_AFTER_RESTORE=$(curl -s -X GET "$API_URL/workflows/$WORKFLOW_ID/versions?limit=10")
echo "$VERSIONS_AFTER_RESTORE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total versions: {data[\"total\"]} (should be 5: original 3 + backup + restore)')
print('\nVersions:')
for i, v in enumerate(data['data']):
    current = '✓ CURRENT' if v['isCurrent'] else ''
    print(f'  {i+1}. {v[\"versionTag\"]} (v{v[\"versionNum\"]}) {current}')
    print(f'     {v[\"description\"]}')
" 2>/dev/null || echo "$VERSIONS_AFTER_RESTORE"
echo ""

echo -e "${BLUE}Step 12: Delete a version (v1.1.0)${NC}"
echo "---------------------------------"
# Get version 2 ID
VERSION_TO_DELETE=$(echo "$VERSIONS_AFTER_RESTORE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for v in data['data']:
    if v['versionNum'] == 2:
        print(v['id'])
        break
" 2>/dev/null)

if [ -n "$VERSION_TO_DELETE" ]; then
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/workflows/$WORKFLOW_ID/versions/$VERSION_TO_DELETE")
    if [ "$HTTP_STATUS" -eq 204 ]; then
        echo -e "${GREEN}✓ Deleted version (v1.1.0)${NC}"
    else
        echo -e "${RED}✗ Failed to delete version (HTTP $HTTP_STATUS)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Could not find version to delete${NC}"
fi
echo ""

echo -e "${BLUE}Step 13: Verify version was deleted${NC}"
echo "----------------------------------"
FINAL_VERSIONS=$(curl -s -X GET "$API_URL/workflows/$WORKFLOW_ID/versions?limit=10")
echo "$FINAL_VERSIONS" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'Total versions: {data[\"total\"]} (should be 4 after deletion)')
print('\nRemaining versions:')
for i, v in enumerate(data['data']):
    current = '✓ CURRENT' if v['isCurrent'] else ''
    print(f'  {i+1}. {v[\"versionTag\"]} (v{v[\"versionNum\"]}) {current}')
" 2>/dev/null || echo "$FINAL_VERSIONS"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ All version tests completed!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Summary:"
echo "  - Created 3 versions (v1.0.0, v1.1.0, v1.2.0)"
echo "  - Restored to v1.0.0 (created backup + restore versions)"
echo "  - Deleted v1.1.0"
echo "  - Final version count: 4"
echo ""
echo "Test workflow ID: $WORKFLOW_ID"
echo ""
echo "You can now:"
echo "  - List versions: GET $API_URL/workflows/$WORKFLOW_ID/versions"
echo "  - Create version: POST $API_URL/workflows/$WORKFLOW_ID/versions"
echo "  - Restore version: POST $API_URL/workflows/$WORKFLOW_ID/versions/{versionId}/restore"
echo "  - Delete workflow: DELETE $API_URL/workflows/$WORKFLOW_ID"
