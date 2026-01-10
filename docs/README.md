# m9m Documentation

Documentation for m9m, a high-performance workflow automation platform built in Go.

## Documentation Structure

### Getting Started
- [Technical Specification](technical-spec.md) - System overview and architecture
- [Performance Report](performance-report.md) - Benchmarks and optimization

### Architecture
- [Architecture Overview](architecture/README.md) - System design
- [Distributed Architecture](architecture/DISTRIBUTED_ARCHITECTURE.md) - Multi-node setup
- [Cluster Implementation](architecture/CLUSTER_IMPLEMENTATION.md) - Clustering details
- [Hybrid Architecture](architecture/HYBRID_ARCHITECTURE.md) - Deployment patterns
- [Scalability Analysis](architecture/SCALABILITY_ANALYSIS.md) - Scaling considerations

### API
- [API Compatibility](api/API_COMPATIBILITY.md) - n8n API compatibility
- [API Implementation](api/API_IMPLEMENTATION_SUMMARY.md) - Endpoint documentation
- [Licensing & API](api/API_AND_LICENSING.md) - API access and licensing

### Deployment
- [Deployment Guide](deployment/DEPLOYMENT_GUIDE.md) - Getting started with deployment
- [Production Guide](deployment/production.md) - Production-ready configuration
- [Deployment Overview](deployment/README.md) - Deployment strategies

### SDK & Language Bindings
- [SDK Overview](sdk/README.md) - Embedding m9m in your applications
- Go SDK (`pkg/m9m`) - Native Go library
- Python Bindings (`bindings/python`) - ctypes-based Python library
- Node.js Bindings (`bindings/nodejs`) - N-API native addon

### Features
- [Plugin System](PLUGIN_SYSTEM.md) - Plugin development and usage
- [Plugin Architecture](PLUGIN_ARCHITECTURE.md) - Internal plugin design
- [Workflow Versions](WORKFLOW_VERSIONS.md) - Versioning workflows
- [Variables & Environments](VARIABLES_AND_ENVIRONMENTS.md) - Configuration management

### Nodes
- [Node Overview](nodes/README.md) - Node system documentation
- [Transform Nodes](nodes/transform/) - Data transformation nodes
- [Trigger Nodes](nodes/trigger/) - Event trigger nodes

### Operations
- [Monitoring](monitoring/README.md) - Metrics and observability
- [Security Review](security/security-review.md) - Security considerations
- [Migration from n8n](migration/from-n8n.md) - Migration guide

### Reference
- [n8n Feature Comparison](N8N_FEATURE_COMPARISON.md) - Feature parity matrix
- [Roadmap](roadmap.md) - Future development plans
- [Contributing](CONTRIBUTING.md) - How to contribute

## Key Advantages

### Performance
- **Startup Time**: < 500ms vs 3s for n8n
- **Memory Usage**: 150MB vs 512MB for n8n
- **Execution Speed**: 5-10x faster workflow execution
- **Container Size**: 300MB vs 1.2GB for n8n

### Enterprise Features
- Built-in Prometheus metrics
- OpenTelemetry distributed tracing
- Horizontal scaling with queue systems
- Git-based workflow versioning

## License

m9m is released under the Apache 2.0 License.
