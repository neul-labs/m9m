# Contributing to m9m

We welcome contributions from the community! This guide will help you get started with contributing to m9m, whether you're fixing bugs, adding features, or improving documentation.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Code Standards](#code-standards)
4. [Testing Guidelines](#testing-guidelines)
5. [Pull Request Process](#pull-request-process)
6. [Node Development](#node-development)
7. [Documentation](#documentation)
8. [Community Guidelines](#community-guidelines)

## Getting Started

### Prerequisites

- **Go 1.21+**: Latest stable version recommended
- **Git**: For version control
- **Make**: For build automation
- **Docker**: For containerized testing (optional)
- **Redis**: For queue testing (optional)

### Ways to Contribute

1. **Bug Reports**: Report issues or unexpected behavior
2. **Feature Requests**: Suggest new features or improvements
3. **Code Contributions**: Fix bugs or implement features
4. **Node Development**: Create new workflow nodes
5. **Documentation**: Improve guides, examples, and API docs
6. **Testing**: Add test cases or improve test coverage

## Development Setup

### 1. Fork and Clone
```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/your-username/m9m.git
cd m9m

# Add upstream remote
git remote add upstream https://github.com/neul-labs/m9m.git
```

### 2. Install Dependencies
```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools

# Verify installation
go version
make --version
```

### 3. Build and Test
```bash
# Build the application
make build

# Run tests
make test

# Run integration tests
make test-integration

# Start development server
make dev
```

### 4. Development Environment
```bash
# Copy example configuration
cp config.example.yaml config.yaml

# Set environment variables
export M9M_DEV_MODE=true
export M9M_LOG_LEVEL=debug

# Start with file watching
make dev-watch
```

## Code Standards

### Go Code Style

We follow standard Go conventions and additional project-specific guidelines:

#### 1. Formatting
```bash
# Format code with gofmt
make fmt

# Run linting
make lint

# Fix common issues
make fix
```

#### 2. Naming Conventions
- **Packages**: lowercase, single word when possible
- **Types**: PascalCase (e.g., `WorkflowEngine`, `NodeMetadata`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase for local, PascalCase for exported
- **Constants**: PascalCase or ALL_CAPS for package-level

#### 3. Code Structure
```go
// Package declaration
package example

// Imports (standard library first, then third-party, then internal)
import (
    "context"
    "fmt"

    "github.com/external/package"

    "github.com/m9m/internal/interfaces"
)

// Constants
const DefaultTimeout = 30 * time.Second

// Types
type ExampleNode struct {
    *base.BaseNode
    config ExampleConfig
}

// Constructor
func NewExampleNode(config ExampleConfig) interfaces.Node {
    return &ExampleNode{
        BaseNode: base.NewBaseNode("Example"),
        config:   config,
    }
}

// Methods (receiver methods grouped together)
func (n *ExampleNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
    // Implementation
}
```

#### 4. Error Handling
```go
// Use specific error types
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
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

    // Continue processing...
    return nil
}
```

#### 5. Comments and Documentation
```go
// Package comment explains the purpose
// Package example provides example functionality for m9m workflows.
package example

// ExampleNode represents a workflow node that demonstrates best practices.
// It implements the interfaces.Node interface and provides example functionality.
type ExampleNode struct {
    config ExampleConfig
}

// Execute processes the workflow execution for this node.
// It validates input parameters, performs the configured operation,
// and returns the results formatted for the workflow engine.
func (n *ExampleNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
    // Implementation details...
}
```

### Directory Structure
```
internal/
├── engine/          # Core workflow execution engine
├── interfaces/      # Shared interfaces and types
├── nodes/          # Node implementations
│   ├── base/       # Base node functionality
│   ├── messaging/  # Messaging platform nodes
│   ├── database/   # Database nodes
│   └── ...         # Other node categories
├── queue/          # Queue system implementations
├── monitoring/     # Monitoring and observability
└── credentials/    # Credential management

cmd/
└── m9m/        # Main application entry point

docs/              # Documentation
tests/             # Integration tests
examples/          # Example workflows and configurations
```

## Testing Guidelines

### 1. Test Structure
```go
package example

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExampleNode_Execute(t *testing.T) {
    tests := []struct {
        name      string
        input     map[string]interface{}
        expected  interface{}
        expectErr bool
    }{
        {
            name: "successful execution",
            input: map[string]interface{}{
                "message": "test message",
            },
            expected: map[string]interface{}{
                "result": "processed: test message",
            },
            expectErr: false,
        },
        {
            name: "missing required parameter",
            input: map[string]interface{}{},
            expected: nil,
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            node := NewExampleNode(ExampleConfig{})
            params := &mockExecutionParams{data: tt.input}

            result, err := node.Execute(context.Background(), params)

            if tt.expectErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.Data)
        })
    }
}
```

### 2. Test Categories

#### Unit Tests
```bash
# Run unit tests
make test-unit

# Run with coverage
make test-cover

# Generate coverage report
make test-cover-html
```

#### Integration Tests
```bash
# Run integration tests (requires Docker)
make test-integration

# Run specific integration test
go test -tags=integration ./tests/integration/database_test.go
```

#### Performance Tests
```bash
# Run benchmark tests
make benchmark

# Profile performance
make profile-cpu
make profile-memory
```

### 3. Mock and Test Utilities
```go
// Use interfaces for testability
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Create mock implementations
type mockHTTPClient struct {
    responses map[string]*http.Response
    errors    map[string]error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    if err, ok := m.errors[req.URL.String()]; ok {
        return nil, err
    }
    return m.responses[req.URL.String()], nil
}
```

## Pull Request Process

### 1. Before You Start
- Check existing issues for similar work
- Create an issue to discuss major changes
- Fork the repository and create a feature branch

### 2. Branch Naming
- `feature/description` - for new features
- `fix/issue-number-description` - for bug fixes
- `docs/description` - for documentation changes
- `refactor/description` - for code refactoring

### 3. Commit Messages
Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): description

feat(nodes): add Stripe payment node
fix(queue): resolve Redis connection timeout
docs(api): update authentication examples
test(database): add MongoDB integration tests
refactor(engine): improve error handling
```

### 4. Pull Request Checklist
- [ ] Tests pass locally (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation is updated
- [ ] Changelog is updated (for significant changes)
- [ ] PR description explains the change
- [ ] Screenshots included for UI changes

### 5. PR Template
```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Screenshots (if applicable)
Include screenshots for UI changes.

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

## Node Development

### 1. Node Structure
```go
package mynodes

import (
    "context"
    "github.com/m9m/internal/interfaces"
    "github.com/m9m/internal/nodes/base"
)

type MyCustomNode struct {
    *base.BaseNode
    httpClient HTTPClient
}

func NewMyCustomNode() interfaces.Node {
    return &MyCustomNode{
        BaseNode:   base.NewBaseNode("MyCustom"),
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (n *MyCustomNode) GetMetadata() interfaces.NodeMetadata {
    return interfaces.NodeMetadata{
        Name:        "My Custom Node",
        Version:     "1.0.0",
        Description: "Custom node for specific functionality",
        Category:    "Custom",
        Icon:        "custom-icon",
        Properties: []interfaces.NodeProperty{
            {
                Name:        "apiKey",
                Type:        "string",
                DisplayName: "API Key",
                Required:    true,
                Secret:      true,
            },
            {
                Name:        "endpoint",
                Type:        "string",
                DisplayName: "API Endpoint",
                Required:    true,
                Default:     "https://api.example.com",
            },
        },
        Credentials: []interfaces.CredentialType{
            {
                Name: "myCustomApi",
                Type: "api",
            },
        },
    }
}

func (n *MyCustomNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
    // Get parameters
    apiKey := params.GetString("apiKey")
    endpoint := params.GetString("endpoint")

    // Validate required parameters
    if apiKey == "" {
        return nil, fmt.Errorf("apiKey is required")
    }

    // Execute logic
    result, err := n.callAPI(ctx, endpoint, apiKey)
    if err != nil {
        return nil, fmt.Errorf("API call failed: %w", err)
    }

    return &base.NodeOutput{
        Data: result,
    }, nil
}
```

### 2. Node Testing
```go
func TestMyCustomNode_Execute(t *testing.T) {
    mockClient := &mockHTTPClient{
        responses: map[string]*http.Response{
            "https://api.example.com/data": {
                StatusCode: 200,
                Body:       ioutil.NopCloser(strings.NewReader(`{"status":"success"}`)),
            },
        },
    }

    node := &MyCustomNode{
        BaseNode:   base.NewBaseNode("MyCustom"),
        httpClient: mockClient,
    }

    params := &mockExecutionParams{
        data: map[string]interface{}{
            "apiKey":   "test-key",
            "endpoint": "https://api.example.com/data",
        },
    }

    result, err := node.Execute(context.Background(), params)

    require.NoError(t, err)
    assert.NotNil(t, result.Data)
}
```

### 3. Node Registration
Add your node to the registry in `cmd/m9m/main.go`:

```go
func registerNodeTypes(engine engine.WorkflowEngine) {
    // ... existing nodes ...

    // Register custom nodes
    myCustomNode := mynodes.NewMyCustomNode()
    engine.RegisterNodeExecutor("n8n-nodes-base.myCustom", myCustomNode)
}
```

## Documentation

### 1. Code Documentation
- All public functions must have comments
- Include examples in complex function comments
- Document package purpose and usage

### 2. User Documentation
- Update relevant guides when adding features
- Include examples and common use cases
- Test documentation with fresh eyes

### 3. API Documentation
- Keep OpenAPI specs updated
- Include request/response examples
- Document error codes and messages

## Community Guidelines

### Code of Conduct
We are committed to providing a welcoming and inclusive environment. Please:

- Be respectful and professional
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Respect different viewpoints and experiences

### Communication
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community discussions
- **Pull Requests**: Code review and collaboration

### Getting Help
- Check existing documentation first
- Search GitHub issues for similar problems
- Ask questions in GitHub Discussions
- Provide clear, reproducible examples

## Release Process

### Versioning
We use [Semantic Versioning](https://semver.org/):
- `MAJOR.MINOR.PATCH`
- Breaking changes increment MAJOR
- New features increment MINOR
- Bug fixes increment PATCH

### Release Checklist
1. Update version numbers
2. Update CHANGELOG.md
3. Run full test suite
4. Update documentation
5. Create release notes
6. Tag release
7. Publish binaries

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributors page

## License

By contributing to m9m, you agree that your contributions will be licensed under the Apache 2.0 License.

---

Thank you for contributing to m9m! Your efforts help make workflow automation better for everyone.
