# n8n-go Compatibility Specification

This document defines the compatibility requirements for n8n-go to ensure it can run exported n8n workflows without modification.

## JSON Workflow Format Compatibility

### Workflow Structure

The exported n8n workflow JSON must be parseable by n8n-go without changes. The structure includes:

```json
{
  "id": "workflow-id",
  "name": "Workflow Name",
  "active": true,
  "nodes": [...],
  "connections": {...},
  "settings": {...},
  "staticData": {...},
  "pinData": {...}
}
```

### Node Structure Compatibility

Each node in the workflow must maintain exact compatibility with n8n's structure:

```json
{
  "parameters": {
    // Node-specific parameters exactly as exported by n8n
  },
  "id": "node-id",
  "name": "Node Name",
  "type": "n8n-nodes-base.httpRequest",
  "typeVersion": 1,
  "position": [250, 300]
}
```

### Connection Structure Compatibility

Connection definitions must match n8n's format exactly:

```json
{
  "Source Node": {
    "main": [
      [
        {
          "node": "Target Node",
          "type": "main",
          "index": 0
        }
      ]
    ]
  }
}
```

## Node Implementation Compatibility

### HTTP Request Node

Must support all parameters from n8n's HTTP Request node:
- `url`: Request URL
- `method`: HTTP method
- `authentication`: Authentication type
- `headers`: HTTP headers
- `parameters`: Query parameters
- `body`: Request body
- `options`: Additional options
- `responseFormat`: Response format

Output must match n8n's format:
```json
{
  "json": {
    "statusCode": 200,
    "headers": {...},
    "body": {...}
  }
}
```

### Set Node

Must support:
- `assignments`: Value assignments
- `options`: Node options

### Function Node

Must support:
- `jsCode`: JavaScript code (will need Go equivalent)
- `parameters`: Function parameters

### Item Lists Node

Must support:
- `values`: List values
- `options`: Node options

## Data Format Compatibility

### Input/Output Data Structure

All nodes must accept and produce data in the format:
```go
type DataItem struct {
    JSON    map[string]interface{} `json:"json"`
    Binary  map[string]BinaryData  `json:"binary,omitempty"`
    PairedItem interface{}           `json:"pairedItem,omitempty"`
}
```

### Binary Data Handling

Binary data must be handled identically:
- Base64 encoding/decoding
- MIME type preservation
- File size tracking
- File name preservation

### Paired Item Tracking

Paired item tracking must maintain compatibility:
- Item indices
- Input source tracking
- Proper propagation through workflow

## Expression Compatibility

### Expression Syntax

Support n8n's expression syntax:
- `{{ $json.property }}`
- `{{ $parameter.property }}`
- `{{ $input[0].json.property }}`

### Built-in Functions

Support all n8n built-in functions:
- String manipulation functions
- Date/time functions
- Mathematical functions
- Array/object manipulation
- Flow control functions

### Variable Resolution

Variable resolution must match n8n exactly:
- `$json`: Current item JSON
- `$parameter`: Node parameters
- `$input`: Input data
- `$execution`: Execution context
- `$workflow`: Workflow context
- `$node`: Node references

## Credential Compatibility

### Credential Structure

Credential handling must support:
- Same credential types as n8n
- Identical parameter structures
- Secure storage and retrieval
- Environment variable substitution

### Supported Credential Types

Must support major credential types:
- HTTP Basic Auth
- API Key
- OAuth1
- OAuth2
- AWS
- Google
- Azure
- And others commonly used in n8n

## Settings Compatibility

### Workflow Settings

Support all workflow settings:
- Timezone
- Execution order
- Error handling
- Timeout settings
- Save data settings

### Node Settings

Support node-specific settings:
- Retry configuration
- Timeout settings
- Error behavior
- Output options

## Error Handling Compatibility

### Error Format

Errors must be reported in a compatible format:
- Same error types
- Identical error messages
- Consistent error data structure

### Error Propagation

Error propagation must match n8n:
- Continue on fail behavior
- Retry mechanisms
- Error recovery options

## Webhook Compatibility

### Webhook Structure

Webhook handling must support:
- Same URL structure
- Identical payload format
- Compatible authentication
- Same response handling

### Trigger Compatibility

Trigger nodes must behave identically:
- Same triggering conditions
- Identical payload structure
- Compatible scheduling
- Same error handling

## Testing Compatibility

### Test Data (Pin Data)

Support pin data exactly as n8n:
- Same data structure
- Identical behavior
- Compatible with all node types

### Execution Results

Execution results must match n8n:
- Same output format
- Identical data structures
- Compatible with downstream processing

## Version Compatibility

### Type Versions

Support versioned node types:
- Backward compatibility with older versions
- Forward compatibility with newer versions
- Graceful handling of unknown versions

### Workflow Versions

Handle workflow versioning:
- Version ID tracking
- Change detection
- Compatibility layers

## Migration Compatibility

### No Migration Required

n8n-go must run exported workflows without:
- Format conversion
- Manual adjustments
- Compatibility layers
- Special configuration

## Validation Compatibility

### Workflow Validation

Validation must match n8n:
- Same validation rules
- Identical error messages
- Compatible with n8n's validation logic

### Node Validation

Node validation must be identical:
- Parameter validation
- Connection validation
- Credential validation
- Settings validation