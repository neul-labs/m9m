# n8n-go Plugin System

The n8n-go plugin system allows you to extend n8n-go with custom nodes without recompiling the core application. This provides flexibility to add domain-specific functionality, integrate with proprietary systems, or rapidly prototype new nodes.

## Supported Plugin Types

n8n-go supports three types of plugins:

1. **JavaScript Plugins** - Write nodes in JavaScript using a familiar Node.js-like API
2. **gRPC Plugins** - Connect to external services via gRPC for distributed node execution
3. **REST API Plugins** - Connect to external services via REST APIs for maximum compatibility

## Quick Start

### Loading Plugins

Start n8n-go with the `--plugin-dir` flag to specify the directory containing your plugins:

```bash
./n8n-go --plugin-dir ./plugins/examples
```

The plugin system will automatically discover and load all plugins in the directory:
- `*.js` files - JavaScript plugins
- `*.grpc.yaml` files - gRPC plugin configurations
- `*.rest.yaml` files - REST API plugin configurations

### Example Output

```
Loading plugins from directory: ./plugins/examples
✅ Loaded JavaScript plugin: textTransform
✅ Loaded gRPC plugin: sentimentAnalysis
✅ Loaded REST plugin: geocoder
✅ Loaded 3 plugins
Registered plugin node: n8n-nodes-base.textTransform
Registered plugin node: n8n-nodes-base.sentimentAnalysis
Registered plugin node: n8n-nodes-base.geocoder
Registered 18 node types
```

## JavaScript Plugins

JavaScript plugins are the most flexible and easiest to get started with. They run directly in the Go application using the Goja JavaScript runtime.

### Basic Structure

```javascript
module.exports = {
    // Node metadata
    description: {
        displayName: "My Custom Node",
        name: "myCustomNode",
        description: "Does something amazing",
        category: "transform",
        icon: "fa:magic",
        inputs: 1,
        outputs: 1,
        properties: [
            {
                displayName: "Operation",
                name: "operation",
                type: "string",
                default: "process"
            }
        ]
    },

    // Main execution function
    execute: function(inputData, parameters) {
        var outputData = [];

        for (var i = 0; i < inputData.length; i++) {
            var item = inputData[i];
            // Process item.json
            var result = {
                json: {
                    processed: true,
                    original: item.json
                }
            };
            outputData.push(result);
        }

        return outputData;
    },

    // Optional validation function
    validate: function(parameters) {
        if (!parameters.operation) {
            return "Operation is required";
        }
        return null; // null = validation passed
    }
};
```

### Available JavaScript Features

#### Console Logging
```javascript
console.log("Info message");
console.error("Error message");
console.warn("Warning message");
```

#### Data Structure

**Input Data:**
```javascript
[
    {
        json: {
            field1: "value1",
            field2: 123
        },
        binary: {
            file: {
                data: "base64-encoded-data",
                mimeType: "image/png",
                fileName: "image.png"
            }
        }
    }
]
```

**Parameters:**
```javascript
{
    operation: "uppercase",
    fieldName: "text",
    customValue: 42
}
```

#### Example: Text Transform

See [examples/textTransform.js](examples/textTransform.js) for a complete working example.

### Best Practices for JavaScript Plugins

1. **Always return an array** - Even for single items, return `[item]`
2. **Preserve binary data** - Copy `item.binary` to output if present
3. **Handle errors gracefully** - Use try-catch and return meaningful errors
4. **Use console.log for debugging** - Logs appear in n8n-go output
5. **Validate inputs** - Check parameter types and required fields
6. **Keep it simple** - Complex logic should use gRPC/REST plugins

## gRPC Plugins

gRPC plugins delegate node execution to an external gRPC service. This is ideal for:
- Compute-intensive operations
- Nodes written in other languages (Python, Java, etc.)
- Integration with existing gRPC services
- Enterprise microservice architectures

### Configuration Format

```yaml
name: myGrpcNode
description: My custom gRPC node
category: ai

# gRPC service address
address: localhost:50051

# Optional timeout (default: 30s)
timeout: 30s

# Node parameters
parameters:
  inputText:
    type: string
    required: true
    description: Text to process

  model:
    type: options
    required: false
    default: standard
    options:
      - standard
      - advanced
```

### gRPC Service Interface

Your gRPC service must implement this interface:

```protobuf
service NodeService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  rpc Describe(DescribeRequest) returns (DescribeResponse);
}

message ExecuteRequest {
  string inputData = 1;   // JSON-encoded array of data items
  string parameters = 2;  // JSON-encoded parameters object
}

message ExecuteResponse {
  bool success = 1;
  string outputData = 2;  // JSON-encoded array of result items
  string error = 3;       // Error message if success=false
}

message DescribeRequest {}

message DescribeResponse {
  string name = 1;
  string description = 2;
  string category = 3;
}
```

### Example: Python gRPC Service

```python
import grpc
import json
from concurrent import futures
import nodeservice_pb2
import nodeservice_pb2_grpc

class NodeServiceServicer(nodeservice_pb2_grpc.NodeServiceServicer):
    def Execute(self, request, context):
        # Parse input
        input_data = json.loads(request.inputData)
        parameters = json.loads(request.parameters)

        # Process data
        output_data = []
        for item in input_data:
            result = {
                "json": {
                    "processed": True,
                    "original": item.get("json", {})
                }
            }
            output_data.append(result)

        # Return response
        return nodeservice_pb2.ExecuteResponse(
            success=True,
            outputData=json.dumps(output_data)
        )

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    nodeservice_pb2_grpc.add_NodeServiceServicer_to_server(
        NodeServiceServicer(), server
    )
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

### Example Configuration

See [examples/sentimentAnalysis.grpc.yaml](examples/sentimentAnalysis.grpc.yaml) for a complete example.

## REST API Plugins

REST API plugins are the most compatible option, allowing integration with any HTTP service.

### Configuration Format

```yaml
name: myRestNode
description: My custom REST API node
category: transform

# REST API endpoint
endpoint: http://localhost:8090/api/node/execute

# HTTP method (default: POST)
method: POST

# Optional timeout (default: 30s)
timeout: 30s

# Optional headers
headers:
  X-API-Version: "1.0"
  Authorization: "Bearer ${env:API_TOKEN}"

# Node parameters
parameters:
  address:
    type: string
    required: true
    description: Address to process

  includeDetails:
    type: boolean
    required: false
    default: true
```

### REST API Interface

Your REST service must implement this endpoint:

**Request:**
```http
POST /api/node/execute HTTP/1.1
Content-Type: application/json

{
  "inputData": [
    {
      "json": {"field": "value"}
    }
  ],
  "parameters": {
    "address": "123 Main St",
    "includeDetails": true
  }
}
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "outputData": [
    {
      "json": {
        "latitude": 40.7128,
        "longitude": -74.0060
      }
    }
  ]
}
```

**Error Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": false,
  "error": "Invalid address format"
}
```

### Example: Node.js REST Service

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/api/node/execute', (req, res) => {
    const { inputData, parameters } = req.body;

    try {
        const outputData = inputData.map(item => {
            return {
                json: {
                    processed: true,
                    original: item.json,
                    parameters: parameters
                }
            };
        });

        res.json({
            success: true,
            outputData: outputData
        });
    } catch (error) {
        res.json({
            success: false,
            error: error.message
        });
    }
});

app.listen(8090, () => {
    console.log('REST node service listening on port 8090');
});
```

### Example Configuration

See [examples/geocoder.rest.yaml](examples/geocoder.rest.yaml) for a complete example.

## Choosing the Right Plugin Type

| Feature | JavaScript | gRPC | REST API |
|---------|-----------|------|----------|
| **Ease of Development** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| **Performance** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Language Support** | JavaScript only | Any | Any |
| **Deployment** | Embedded | Separate service | Separate service |
| **Use Case** | Simple transforms | Heavy computation | External APIs |
| **Scalability** | Limited | High | High |
| **Latency** | Very low | Low | Medium |

### Recommendations

- **Use JavaScript** for:
  - Simple data transformations
  - String manipulation
  - JSON processing
  - Quick prototyping
  - Single-server deployments

- **Use gRPC** for:
  - CPU-intensive operations
  - Machine learning inference
  - Complex business logic
  - Services in Python, Java, Go, etc.
  - Low-latency requirements

- **Use REST** for:
  - Integration with existing HTTP APIs
  - Services without gRPC support
  - Maximum compatibility
  - Firewall-friendly deployments

## Plugin Development Workflow

### 1. Create Plugin

Choose your plugin type and create the appropriate file:

```bash
# JavaScript plugin
touch plugins/myNode.js

# gRPC plugin
touch plugins/myNode.grpc.yaml

# REST plugin
touch plugins/myNode.rest.yaml
```

### 2. Implement Logic

Write your plugin code (see examples above).

### 3. Test Plugin

Start n8n-go with your plugin directory:

```bash
./n8n-go --plugin-dir ./plugins
```

Check the logs for successful loading:
```
✅ Loaded JavaScript plugin: myNode
Registered plugin node: n8n-nodes-base.myNode
```

### 4. Use in Workflows

Your plugin node is now available in workflows with the name `n8n-nodes-base.myNode`.

Example workflow:
```json
{
  "nodes": [
    {
      "type": "n8n-nodes-base.myNode",
      "name": "My Node",
      "parameters": {
        "operation": "process"
      }
    }
  ]
}
```

### 5. Debug

Use console.log in JavaScript plugins:
```javascript
console.log("Debug info:", parameters);
```

Check n8n-go logs for plugin errors:
```bash
./n8n-go --plugin-dir ./plugins 2>&1 | grep -i error
```

## Advanced Topics

### Hot Reloading

Plugin hot reloading is supported but requires manual trigger:

```bash
# Send SIGUSR1 to reload plugins
kill -USR1 $(pgrep n8n-go)
```

### Environment Variables

REST and gRPC plugins support environment variable substitution:

```yaml
headers:
  Authorization: "Bearer ${env:API_TOKEN}"
```

Set environment variables before starting:
```bash
export API_TOKEN=your-token-here
./n8n-go --plugin-dir ./plugins
```

### Error Handling

**JavaScript:**
```javascript
execute: function(inputData, parameters) {
    try {
        // Your logic
    } catch (error) {
        console.error("Error:", error.message);
        throw error; // Re-throw to fail the node
    }
}
```

**gRPC/REST:**
Return `success: false` with error message:
```json
{
  "success": false,
  "error": "Detailed error message"
}
```

### Binary Data

JavaScript plugins can handle binary data:

```javascript
execute: function(inputData, parameters) {
    var item = inputData[0];

    // Access binary data
    if (item.binary && item.binary.file) {
        var fileData = item.binary.file.data; // base64 string
        var mimeType = item.binary.file.mimeType;
        console.log("File type:", mimeType);
    }

    // Return with binary data
    return [{
        json: { processed: true },
        binary: item.binary // Preserve binary
    }];
}
```

## Troubleshooting

### Plugin Not Loading

**Check logs:**
```bash
./n8n-go --plugin-dir ./plugins 2>&1 | grep -i "plugin"
```

**Common issues:**
- File extension must be `.js`, `.grpc.yaml`, or `.rest.yaml`
- YAML syntax errors (use `yamllint`)
- JavaScript syntax errors (test with `node`)
- File permissions

### JavaScript Errors

**ReferenceError: X is not defined**
- Limited JavaScript features available
- No Node.js modules (fs, http, etc.)
- Use console.log to debug

**TypeError: Cannot read property**
- Check if `item.json` exists
- Validate parameter types
- Handle undefined values

### gRPC Connection Failed

**Error: failed to connect to gRPC service**
- Verify service is running: `grpc_health_probe -addr=localhost:50051`
- Check firewall rules
- Verify address format: `host:port`

### REST API Errors

**Error: HTTP request failed with status 404**
- Verify endpoint URL
- Check service logs
- Test with curl:
```bash
curl -X POST http://localhost:8090/api/node/execute \
  -H "Content-Type: application/json" \
  -d '{"inputData":[],"parameters":{}}'
```

## Performance Considerations

### JavaScript Plugins

- **Startup**: ~5ms per plugin
- **Execution**: ~1-10ms overhead per call
- **Memory**: ~1-2MB per plugin
- **Concurrency**: Limited by Go runtime

### gRPC Plugins

- **Startup**: ~50-100ms (connection establishment)
- **Execution**: ~5-20ms overhead per call
- **Memory**: Minimal (in Go process)
- **Concurrency**: High (separate service)

### REST Plugins

- **Startup**: ~10-20ms (HTTP client creation)
- **Execution**: ~20-50ms overhead per call
- **Memory**: Minimal (in Go process)
- **Concurrency**: High (separate service)

## Security Considerations

1. **JavaScript plugins run in-process** - They have access to the same resources as n8n-go
2. **Validate all inputs** - Never trust user-provided parameters
3. **Use HTTPS for REST APIs** - Encrypt data in transit
4. **Use TLS for gRPC** - Enable secure connections in production
5. **Limit plugin directory permissions** - Only load trusted plugins
6. **Review plugin code** - Audit JavaScript plugins before loading

## Examples

All examples are in the `examples/` directory:

- [textTransform.js](examples/textTransform.js) - JavaScript plugin for text transformation
- [sentimentAnalysis.grpc.yaml](examples/sentimentAnalysis.grpc.yaml) - gRPC sentiment analysis
- [geocoder.rest.yaml](examples/geocoder.rest.yaml) - REST geocoding service

## Additional Resources

- [Plugin Architecture Documentation](../docs/PLUGIN_ARCHITECTURE.md)
- [Goja JavaScript Runtime](https://github.com/dop251/goja)
- [gRPC Documentation](https://grpc.io/docs/)
- [n8n Node Development](https://docs.n8n.io/integrations/creating-nodes/)

## Contributing

We welcome plugin contributions! To share your plugins:

1. Test your plugin thoroughly
2. Add documentation and examples
3. Submit a pull request to the [n8n-go-plugins](https://github.com/dipankar/n8n-go-plugins) repository

## Support

For plugin-related questions:
- Open an issue on GitHub
- Join our Discord community
- Check the documentation

---

**Happy plugin development!** 🚀
