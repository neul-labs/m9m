# Contributing to n8n-go

This document provides guidelines for contributing to the n8n-go project.

## Project Overview

n8n-go is a reimplementation of the n8n workflow automation platform in Go, designed for better performance and resource efficiency while maintaining 100% compatibility with exported n8n workflows.

## Getting Started

### Prerequisites

- Go 1.19 or later
- Git
- Basic understanding of workflow automation concepts

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/n8n-go.git
   cd n8n-go
   ```

3. Verify you can build the project:
   ```bash
   go build ./cmd/n8n-go
   ```

### Project Structure

```
n8n-go/
├── cmd/
│   └── n8n-go/           # Main application entry point
├── internal/
│   ├── engine/           # Core workflow execution engine
│   ├── nodes/            # Node type implementations
│   ├── model/            # Data structures and JSON handling
│   ├── connections/      # Data routing between nodes
│   ├── expressions/      # Expression evaluation system
│   ├── credentials/      # Credential management
│   └── utils/            # Helper functions
├── docs/                 # Project documentation
├── go.mod                # Go module definition
└── README.md             # Project overview
```

## Development Process

### 1. Review Documentation

Before starting development, review:
- [Project Plan](docs/project-plan.md)
- [Technical Specification](docs/technical-spec.md)
- [Roadmap](docs/roadmap.md)
- [Design Document](docs/design.md)

### 2. Choose an Area to Work On

Check the [Roadmap](docs/roadmap.md) for planned features and current phase.

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 4. Implement Your Changes

Follow Go best practices:
- Write clean, readable code
- Add comprehensive unit tests
- Follow existing code patterns
- Document public APIs

### 5. Test Your Changes

```bash
# Run unit tests
go test ./...

# Build and test the application
go build ./cmd/n8n-go
./n8n-go
```

### 6. Submit a Pull Request

- Ensure all tests pass
- Update documentation if needed
- Follow the pull request template

## Code Standards

### Go Coding Standards

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format code
- Use `golint` and `go vet` to check code quality
- Write meaningful commit messages

### Testing

- Write unit tests for all new functionality
- Aim for 80%+ test coverage
- Use table-driven tests where appropriate
- Test error conditions

### Documentation

- Document all public functions and types
- Update relevant documentation files
- Add examples for complex functionality

## Areas Needing Contribution

### Core Implementation

1. **Data Models** - Implement JSON parsing and data structures
2. **Workflow Engine** - Core execution engine
3. **Node Interface** - Base node interface and utilities
4. **Connection System** - Data routing between nodes

### Node Types

1. **HTTP Request** - Core HTTP integration node
2. **Data Transformation** - Set, Item Lists, Function nodes
3. **Database Nodes** - SQL query and database operations
4. **File Operations** - Read/Write binary files
5. **Communication** - Email, Slack, webhook nodes
6. **Triggers** - Cron, wait, and trigger nodes

### Infrastructure

1. **Expression Engine** - n8n-style expression evaluation
2. **Credential Management** - Secure credential storage
3. **CLI Interface** - Command-line tools
4. **Webhook Support** - Webhook server and handling

## Communication

- Join our development discussions in the issues
- For major changes, create an issue first to discuss the approach
- Be respectful and inclusive in all interactions

## License

By contributing to n8n-go, you agree that your contributions will be licensed under the Apache 2.0 License.