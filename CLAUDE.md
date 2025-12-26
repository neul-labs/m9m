# CLAUDE.md

This file provides comprehensive guidance to Claude Code when working with the m9m repository - a high-performance workflow automation platform built in Go.

## Project Overview

m9m is a cloud-native workflow automation platform that provides 95% backend feature parity with n8n while offering significant performance improvements. It's built using Go for maximum performance and minimal resource usage, targeting enterprise and developer environments.

**Key Characteristics:**
- **Performance**: 5-10x faster execution than n8n
- **Memory Efficiency**: 70% lower memory usage (~150MB vs 512MB)
- **Container Size**: 75% smaller containers (300MB vs 1.2GB)
- **Startup Time**: Sub-second startup (500ms vs 3s)
- **Architecture**: Cloud-native, microservices-ready

## Technology Stack

- **Language**: Go 1.21+
- **Architecture**: Modular, interface-driven design
- **Queue Systems**: Memory, Redis, RabbitMQ support
- **Monitoring**: Prometheus metrics, OpenTelemetry tracing
- **Testing**: Go standard testing, testify library
- **Build System**: Make-based build automation
- **Deployment**: Docker, Kubernetes-native

## Essential Commands

### Building and Development

Always use the Makefile for consistent builds:

```bash
# Build the application
make build

# Install dependencies
make deps

# Format and validate code
make fmt
make vet
make lint

# Run all tests
make test

# Generate test coverage
make coverage
make coverage-html

# Clean build artifacts
make clean
```

### Testing Commands

Testing should be comprehensive and follow Go conventions:

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/engine/...

# Run with verbose output
go test -v ./internal/nodes/...

# Run with coverage
go test -cover ./...

# Run integration tests (requires Docker)
./bin/integration-test

# Run benchmarks
./bin/benchmark
```

### Code Quality

Always ensure code quality before committing:

```bash
# Format code (required before commit)
go fmt ./...

# Vet code for common issues
go vet ./...

# Run linter (install golint if needed)
golint ./...

# Run all quality checks
make fmt vet lint test
```

## Project Structure

### Core Architecture

```
m9m/
├── cmd/                    # Application entry points
│   ├── m9m/            # Main application
│   ├── benchmark/         # Performance benchmarking
│   ├── integration-test/  # Integration testing
│   └── template-cli/      # Template management
├── internal/              # Private application code
│   ├── engine/            # Core workflow execution engine
│   ├── nodes/             # Node implementations
│   │   ├── base/          # Base node interfaces and types
│   │   ├── messaging/     # Messaging platform nodes (Slack, Discord, etc.)
│   │   ├── database/      # Database nodes (PostgreSQL, MongoDB, etc.)
│   │   ├── cloud/         # Cloud platform nodes (AWS, GCP, Azure)
│   │   ├── ai/            # AI/LLM nodes (OpenAI, Anthropic)
│   │   ├── transform/     # Data transformation nodes
│   │   └── trigger/       # Trigger nodes (webhooks, timers)
│   ├── monitoring/        # Prometheus metrics and observability
│   ├── credentials/       # Credential management system
│   ├── expressions/       # Expression evaluation engine
│   ├── runtime/           # JavaScript and Python runtime support
│   ├── api/               # REST API implementation
│   ├── queue/             # Queue system implementations
│   └── compatibility/     # n8n compatibility layer
├── docs/                  # Comprehensive documentation
├── examples/              # Example workflows and configurations
├── test-workflows/        # Test workflow definitions
└── Makefile               # Build automation
```

### Key Interfaces and Patterns

#### Node Implementation Pattern

All nodes must implement the `NodeExecutor` interface:

```go
type NodeExecutor interface {
    Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error)
    Description() NodeDescription
    ValidateParameters(params map[string]interface{}) error
}
```

#### Base Node Usage

Always extend `BaseNode` for common functionality:

```go
type CustomNode struct {
    *base.BaseNode
    // Custom fields
}

func NewCustomNode() *CustomNode {
    return &CustomNode{
        BaseNode: base.NewBaseNode(NodeDescription{
            Name:        "Custom Node",
            Description: "Does custom processing",
            Category:    "transform",
        }),
    }
}
```

#### Registration Pattern

Nodes are registered in `cmd/m9m/main.go`:

```go
func registerNodeTypes(engine engine.WorkflowEngine) {
    customNode := mynodes.NewCustomNode()
    engine.RegisterNodeExecutor("n8n-nodes-base.custom", customNode)
}
```

## Development Guidelines

### Code Standards

#### Naming Conventions
- **Packages**: lowercase, single word when possible
- **Types**: PascalCase (e.g., `WorkflowEngine`, `NodeMetadata`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase for local, PascalCase for exported
- **Constants**: PascalCase or ALL_CAPS for package-level

#### Error Handling
```go
// Use specific error types
type ValidationError struct {
    Field   string
    Message string
}

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to execute node %s: %w", nodeName, err)
}

// Use early returns
func processData(data []byte) error {
    if len(data) == 0 {
        return errors.New("data cannot be empty")
    }
    return nil
}
```

#### Documentation
```go
// Package comments explain purpose
// Package example provides workflow node functionality.
package example

// ExampleNode represents a workflow node for data processing.
type ExampleNode struct {
    config ExampleConfig
}

// Execute processes workflow data according to node configuration.
func (n *ExampleNode) Execute(ctx context.Context, data []model.DataItem) ([]model.DataItem, error) {
    // Implementation
}
```

### Testing Standards

#### Test Structure
```go
func TestNodeName_Execute(t *testing.T) {
    tests := []struct {
        name      string
        input     []model.DataItem
        params    map[string]interface{}
        expected  []model.DataItem
        expectErr bool
    }{
        {
            name: "successful execution",
            input: []model.DataItem{{JSON: map[string]interface{}{"test": "data"}}},
            params: map[string]interface{}{"param": "value"},
            expected: []model.DataItem{{JSON: map[string]interface{}{"result": "processed"}}},
            expectErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            node := NewNodeName()
            result, err := node.Execute(tt.input, tt.params)

            if tt.expectErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### Mock Usage
```go
// Use interfaces for testability
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Create mock implementations
type mockHTTPClient struct {
    responses map[string]*http.Response
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    return m.responses[req.URL.String()], nil
}
```

### File Organization

#### Package Structure
- Keep packages focused and cohesive
- Use internal/ for private implementation details
- Export only necessary interfaces and types
- Group related functionality in packages

#### Import Organization
```go
import (
    // Standard library first
    "context"
    "fmt"
    "net/http"

    // Third-party packages
    "github.com/external/package"

    // Internal packages last
    "github.com/yourusername/m9m/internal/base"
    "github.com/yourusername/m9m/internal/model"
)
```

## Common Development Tasks

### Adding a New Node

1. **Create node package** in appropriate category:
   ```bash
   mkdir -p internal/nodes/category/newnode
   ```

2. **Implement NodeExecutor interface**:
   ```go
   type NewNode struct {
       *base.BaseNode
   }

   func NewNewNode() *NewNode {
       return &NewNode{
           BaseNode: base.NewBaseNode(NodeDescription{...}),
       }
   }
   ```

3. **Add comprehensive tests**:
   ```bash
   touch internal/nodes/category/newnode/newnode_test.go
   ```

4. **Register in main.go**:
   ```go
   newNode := newnode.NewNewNode()
   engine.RegisterNodeExecutor("n8n-nodes-base.newNode", newNode)
   ```

5. **Add example workflow**:
   ```bash
   touch examples/category/newnode-example.json
   ```

### Debugging Workflows

1. **Enable debug logging**:
   ```bash
   export M9M_LOG_LEVEL=debug
   ```

2. **Run with test data**:
   ```bash
   ./m9m execute test-workflows/debug-workflow.json
   ```

3. **Use test workflows** in `test-workflows/` directory

4. **Check performance** with benchmarks:
   ```bash
   ./bin/benchmark --workflow examples/performance-test.json
   ```

### Performance Optimization

1. **Profile performance**:
   ```bash
   go test -bench=. -cpuprofile cpu.prof
   go tool pprof cpu.prof
   ```

2. **Monitor memory usage**:
   ```bash
   go test -bench=. -memprofile mem.prof
   go tool pprof mem.prof
   ```

3. **Use efficient data structures**:
   - Prefer slices over arrays for variable data
   - Use sync.Pool for object reuse
   - Minimize allocations in hot paths

## Integration Patterns

### Credential Management
```go
// Access credentials in nodes
func (n *MyNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    creds := n.credentialManager.GetCredentials("myService")
    apiKey := creds["apiKey"]
    // Use credentials safely
}
```

### Queue Integration
```go
// For scalable execution
type QueueConfig struct {
    Type        string // "memory", "redis", "rabbitmq"
    URL         string
    MaxWorkers  int
}
```

### Monitoring Integration
```go
// Add metrics to custom nodes
func (n *MyNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    start := time.Now()
    defer func() {
        metrics.NodeExecutionDuration.WithLabelValues(n.Description().Name).Observe(time.Since(start).Seconds())
    }()

    // Node execution logic
}
```

## Configuration Management

### Environment Variables
Key environment variables for development:
```bash
# Core configuration
M9M_PORT=8080
M9M_HOST=0.0.0.0
M9M_LOG_LEVEL=info

# Queue configuration
M9M_QUEUE_TYPE=redis
M9M_QUEUE_URL=redis://localhost:6379
M9M_MAX_WORKERS=10

# Monitoring configuration
M9M_METRICS_PORT=9090
M9M_TRACING_ENDPOINT=http://localhost:14268/api/traces

# Development flags
M9M_DEV_MODE=true
M9M_ENABLE_PPROF=true
```

### Configuration Files
Use YAML configuration files in `config/`:
```yaml
server:
  port: 8080
  host: "0.0.0.0"

queue:
  type: "redis"
  url: "redis://localhost:6379"
  max_workers: 10

monitoring:
  metrics_port: 9090
  enable_tracing: true
```

## Deployment Considerations

### Docker Development
```bash
# Build development image
docker build -t m9m:dev .

# Run with development configuration
docker run -p 8080:8080 -v $(pwd)/config:/app/config m9m:dev
```

### Local Development
```bash
# Start with file watching (if available)
make dev-watch

# Or run manually
./m9m serve --config config.yaml
```

## Compatibility with n8n

### Migration Support
- **100% workflow compatibility** - All n8n workflows run unchanged
- **Expression compatibility** - Same expression syntax and functions
- **Node compatibility** - Compatible node parameter structure

### Key Differences
- **Performance**: Significantly faster execution
- **Resource usage**: Lower memory and CPU usage
- **Deployment**: Cloud-native architecture
- **Monitoring**: Built-in enterprise observability

### Testing Compatibility
```bash
# Test n8n workflow compatibility
./cmd/n8n-compat/n8n-compat --workflow n8n-export.json

# Validate workflow execution
./m9m execute --validate n8n-workflow.json
```

## Best Practices

### Security
- Always validate input parameters
- Use credential manager for sensitive data
- Implement proper error handling without exposing internals
- Follow principle of least privilege

### Performance
- Minimize allocations in execution paths
- Use pooling for reusable objects
- Profile regularly and optimize hot paths
- Consider memory usage in long-running processes

### Maintainability
- Write comprehensive tests for all nodes
- Document complex business logic
- Use consistent error handling patterns
- Follow Go idioms and conventions

### Error Handling
- Use specific error types for different failure modes
- Wrap errors with context for debugging
- Log errors appropriately without exposing sensitive data
- Provide meaningful error messages to users

## Troubleshooting Guide

### Common Issues
1. **Build failures**: Check Go version (1.21+ required)
2. **Test failures**: Ensure all dependencies are available
3. **Memory issues**: Check for goroutine leaks in concurrent code
4. **Performance issues**: Profile and identify bottlenecks

### Debug Commands
```bash
# Check application health
curl http://localhost:8080/health

# View metrics
curl http://localhost:9090/metrics

# Check logs
tail -f /var/log/m9m.log

# Test workflow execution
./m9m execute --debug examples/debug-workflow.json
```

## Contributing Guidelines

### Code Quality Checklist
- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] No linting errors (`make lint`)
- [ ] Documentation updated
- [ ] Example workflow provided (for new nodes)
- [ ] Performance impact assessed

### Pull Request Process
1. Create feature branch from main
2. Implement changes with tests
3. Run full test suite
4. Update documentation
5. Create pull request with clear description

## Future Development

### Planned Features
- Web UI integration (Vue.js frontend)
- Advanced workflow templates
- Enhanced node marketplace
- Multi-tenancy support
- Advanced security features

### Architecture Evolution
- Serverless deployment support
- Advanced queue management
- Enhanced monitoring capabilities
- Plugin ecosystem expansion

---

**Important**: Always run `make test` before committing code. The test suite is comprehensive and catches most common issues. For performance-critical changes, also run benchmarks to ensure no regression.

**Note**: This project maintains high compatibility with n8n while providing significant performance improvements. When in doubt about compatibility, test with actual n8n workflows and refer to the compatibility test suite.