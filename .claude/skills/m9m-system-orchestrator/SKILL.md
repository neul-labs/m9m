---
name: m9m-system-orchestrator
description: End-to-end control plane for m9m. Use when a request spans discovery, workflow creation, execution, debugging, plugin/node extension, and lifecycle management.
---

# m9m System Orchestrator

Use this skill when the user asks for outcomes that require multiple m9m surfaces, for example:
- "Build and run an automation pipeline, then harden it."
- "Set up agent workflows and keep them reliable in production."
- "Manage workflows, nodes, and execution state across the whole system."

## Control Loop

1. Discover current capabilities and constraints first.
2. Design or update workflow topology and node contracts.
3. Execute with sync or async mode based on expected duration.
4. Observe status, logs, and node outputs.
5. Recover via retry/cancel/fix.
6. Promote reusable assets (workflows/plugins), then govern lifecycle.

## Skill Routing

- Use `m9m-capability-discovery` for node/workflow/plugin inventory.
- Use `m9m-workflow-factory` for workflow create/update/validate/activate.
- Use `m9m-execution-ops` for execution run/wait/cancel/retry.
- Use `m9m-debug-reliability` for failures and performance regressions.
- Use `m9m-plugin-node-builder` for custom JavaScript/REST nodes.
- Use `m9m-cli-agent-orchestrator` for CLI agent execution with sandboxing.
- Use `m9m-reuse-lifecycle` for duplication, tagging, rollout, and cleanup.

## MCP-First, REST-Fallback

- Prefer MCP tools when connected to `mcp-server`.
- If MCP is unavailable, use REST endpoints in `documentation/docs/api/*.md`.
- Keep the same lifecycle semantics between MCP and REST.

## Guardrails

- Validate before create/update (`workflow_validate`).
- Track `workflowId` and `executionId` explicitly in outputs.
- Treat unauthenticated API access as internal/experimental only.
- Treat Kubernetes deployment files as experimental; prefer single-binary operations for official paths.

