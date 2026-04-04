# Refactor Plan

This document defines the modularization plan for the m9m codebase.

Goals:
- Reduce oversized files and mixed-responsibility modules
- Improve reuse across backend, frontend, CLI, and runtime layers
- Make feature work safer by creating explicit ownership boundaries
- Remove backup and disabled source artifacts that blur the active code path

Non-goals:
- Large behavior rewrites during the initial split
- Package churn without a clear ownership or reuse benefit
- Premature abstraction that hides current domain boundaries

## Refactor Principles

1. Split by responsibility, not by line count alone.
2. Prefer pure helpers for shared logic before adding new service types.
3. Keep APIs stable while moving code; behavior changes happen after file boundaries are clean.
4. Reuse should flow through domain modules, not cross-package copy/paste.
5. Each module should have a clear public surface and an obvious test location.
6. Delete backup artifacts after verifying the active file is the only code path.
7. Do not create new top-level packages unless file-level modularization is insufficient.

## Success Criteria

- No active source file should own unrelated domains.
- API handlers, scheduler execution, runtime bootstrap, and workflow editor state are isolated by concern.
- Shared workflow graph logic is reused by store, canvas, and future tooling.
- Backup files (`*.bak`, `*.old`, `*.disabled`) are removed from active source trees.
- Refactor steps are covered by focused tests around moved logic.

## Phase 0: Safety Rails

- [ ] Freeze current module boundaries in tests before moving code.
- [x] Record baseline for `go test ./...` and frontend checks.
- [x] Identify generated artifacts that should not be hand-edited.
- [x] Confirm backup files are not referenced by scripts, docs, or build steps.

Acceptance:
- Refactor work can be validated incrementally.
- We have a clear distinction between generated code, active code, and dead backup files.

## Phase 1: Source Cleanup

### Backup Artifact Removal

- [x] Remove all `*.bak`, `*.old`, and `*.disabled` files under `internal/`.
- [x] Check `cmd/`, `web/`, `bindings/`, and docs for equivalent dead artifacts.
- [x] Add a simple CI or lint check preventing backup artifacts from re-entering the tree.

Current known cleanup set includes:
- `internal/api/workflow_api.go.old`
- `internal/nodes/**/*.bak`
- `internal/nodes/**/*.old`
- `internal/nodes/**/*.disabled`
- `internal/runtime/python_runtime_embedded.go.disabled`

Acceptance:
- The repository contains one active implementation path per concern.
- Searches and code review are not polluted by dead source variants.

## Phase 2: API Server Decomposition

Primary target:
- `internal/api/server.go`

Problem:
- Route registration, transport helpers, workflow CRUD, execution, schedules, credentials, templates, copilot, DLQ, and health endpoints are mixed in one file.

Target structure:
- `internal/api/server.go`
  Core `APIServer` type, constructors, shared dependencies
- `internal/api/routes.go`
  Route registration only
- `internal/api/responses.go`
  JSON/error helpers, pagination helpers, request parsing helpers
- `internal/api/workflows.go`
  Workflow CRUD and duplicate/apply flows
- `internal/api/executions.go`
  Sync/async execution, jobs, cancellation
- `internal/api/schedules.go`
  Schedule CRUD and history endpoints
- `internal/api/credentials.go`
  Credential response shaping and handlers
- `internal/api/templates.go`
  Built-in template transport layer
- `internal/api/expressions.go`
  Expression evaluation endpoint and input validation
- `internal/api/copilot.go`
  Copilot request/response handling
- `internal/api/dlq.go`
  DLQ endpoints
- `internal/api/health.go`
  Health, readiness, version, metrics, performance

Todo:
- [x] Move route wiring out of `server.go`.
- [x] Move each handler family to its own file.
- [ ] Extract template data from handler code into dedicated template definitions.
- [x] Remove stale tag handlers from `server.go` if tags are now owned elsewhere.
- [x] Keep request/response shapes stable while moving code.

Reuse focus:
- Shared execution request parsing should be reusable across sync and async endpoints.
- Shared workflow cloning/template-instantiation helpers should not live inside handlers.

Acceptance:
- `server.go` becomes a thin bootstrap file.
- Each endpoint family is discoverable in one place.

## Phase 3: Workflow Graph Core

Primary targets:
- `web/src/stores/workflow.ts`
- `web/src/components/workflow/WorkflowCanvas.vue`
- `web/src/views/WorkflowEditor.vue`

Problem:
- Persistence state, editor state, graph mutation logic, and canvas adapters are mixed together.

Target structure:
- `web/src/stores/workflow.ts`
  API-backed workflow list/detail persistence
- `web/src/stores/workflowEditor.ts`
  Selection, dirty state, current draft, editor actions
- `web/src/lib/workflowGraph.ts`
  Pure graph helpers for node/connection add, remove, clone, rename, normalize
- `web/src/composables/useWorkflowCanvas.ts`
  Vue Flow adapter logic
- `web/src/components/workflow/WorkflowCanvas.vue`
  Presentation shell only
- `web/src/components/workflow/WorkflowEditorToolbar.vue`
  Toolbar UI extracted from the editor view
- `web/src/components/workflow/WorkflowSidebarLayout.vue`
  Optional shell if left/right panel logic keeps growing

Todo:
- [x] Split persistence actions from editor mutation logic.
- [x] Move pure graph mutations into `workflowGraph.ts`.
- [x] Fix node removal sequencing while extracting graph helpers.
- [x] Move Vue Flow translation logic out of the component body.
- [x] Extract editor toolbar from `WorkflowEditor.vue`.

Reuse focus:
- Graph helpers should be usable by canvas, copilot actions, import tools, and future undo/redo.
- Editor state should be reusable across alternate workflow editing surfaces.

Acceptance:
- Canvas becomes a thin adapter.
- Graph mutations are testable without Vue/Pinia.

## Phase 4: JavaScript Runtime Split

Primary target:
- `internal/runtime/javascript_runtime.go`

Problem:
- Runtime bootstrap, security policy, npm loading, mocks, timers, built-in modules, and execution are all mixed.

Target structure:
- `internal/runtime/javascript_runtime.go`
  Public runtime type and constructors
- `internal/runtime/runtime_globals.go`
  Node/global/n8n bootstrap
- `internal/runtime/runtime_security.go`
  env-var allow/block policy and package name validation
- `internal/runtime/runtime_loader.go`
  npm package lookup, caching, module loading
- `internal/runtime/runtime_modules.go`
  built-in module factories
- `internal/runtime/runtime_mocks.go`
  axios/lodash/moment/uuid/crypto-js mocks
- `internal/runtime/runtime_timers.go`
  timers and scheduling helpers
- `internal/runtime/runtime_exec.go`
  Execute, ExecuteExpression, context injection

Todo:
- [x] Move security helpers into their own file.
- [x] Move built-in module factories into grouped files.
- [x] Isolate mock package generation from real package loading.
- [x] Keep loader behavior deterministic under offline fallback.
- [x] Create tests around extracted loader and security helpers.

Reuse focus:
- Loader, mocking, and runtime bootstrap should be separable for future runtimes and tests.

Acceptance:
- The runtime package has explicit sub-areas instead of one monolith.

## Phase 5: Scheduler Modularization

Primary target:
- `internal/scheduler/workflow_scheduler.go`

Problem:
- Schedule registry, cron lifecycle, workflow execution, execution history, metrics, cleanup, and serialization are mixed.

Target structure:
- `internal/scheduler/types.go`
- `internal/scheduler/scheduler.go`
- `internal/scheduler/registry.go`
- `internal/scheduler/executor.go`
- `internal/scheduler/history.go`
- `internal/scheduler/metrics.go`
- `internal/scheduler/cleanup.go`
- `internal/scheduler/serialization.go`

Todo:
- [x] Move data types out first.
- [x] Isolate schedule CRUD from execution flow.
- [x] Move history mutation and averaging into dedicated helpers.
- [x] Move metrics collection and cleanup loops into separate files.

Reuse focus:
- Execution history and schedule config helpers should be reusable by API, storage, and future distributed scheduling.

Acceptance:
- Execution flow is readable without scanning CRUD and metrics code.

## Phase 6: Marketplace and Template Reuse

Primary target:
- `internal/templates/marketplace_manager.go`

Problem:
- Repository sync, cache, search, ratings, analytics, scanning, and scheduler behavior are all in one file.

Target structure:
- `internal/templates/marketplace_types.go`
- `internal/templates/marketplace_manager.go`
- `internal/templates/marketplace_sync.go`
- `internal/templates/marketplace_search.go`
- `internal/templates/marketplace_cache.go`
- `internal/templates/marketplace_ratings.go`
- `internal/templates/marketplace_analytics.go`
- `internal/templates/marketplace_security.go`

Todo:
- [x] Move types and constructors out of the manager file.
- [x] Split sync transports from search and filtering.
- [x] Split cache and ratings into standalone units.
- [x] Isolate scanner rules from manager orchestration.

Reuse focus:
- Template search/filter logic should be callable without pulling in HTTP sync or scheduler behavior.

Acceptance:
- Marketplace behavior is composed from reusable pieces rather than one manager file.

## Phase 7: CLI and Command Boundaries

Primary target:
- `cmd/n8n-compat/main.go`

Problem:
- Command registration and implementation are collapsed into one file.

Target structure:
- `cmd/n8n-compat/main.go`
- `cmd/n8n-compat/root.go`
- `cmd/n8n-compat/workflow.go`
- `cmd/n8n-compat/javascript.go`
- `cmd/n8n-compat/nodes.go`
- `cmd/n8n-compat/tests.go`
- `cmd/n8n-compat/helpers.go`

Todo:
- [x] Mirror the cleaner `cmd/m9m/commands` structure.
- [x] Move shared file IO and printing helpers into reusable helpers.
- [x] Separate benchmark/test helpers from user-facing commands.

Reuse focus:
- Command helpers should be reusable across future admin or migration CLIs.

Acceptance:
- Command registration and command logic are no longer interleaved.

## Phase 8: Expression Library Split

Primary targets:
- `internal/expressions/functions.go`
- `internal/expressions/functions_extended.go`

Problem:
- Category-based extension groups exist, but physical file boundaries do not match those groups.

Target structure:
- `internal/expressions/functions_strings.go`
- `internal/expressions/functions_math.go`
- `internal/expressions/functions_arrays.go`
- `internal/expressions/functions_dates.go`
- `internal/expressions/functions_logic.go`
- `internal/expressions/functions_objects.go`
- `internal/expressions/functions_utils.go`
- `internal/expressions/functions_registry.go`

Todo:
- [ ] Split categories into separate files.
- [ ] Centralize registration in one small registry file.
- [ ] Keep category ownership obvious and independently testable.

Reuse focus:
- Function groups become individually testable and easier to extend.

Acceptance:
- New expression functions can be added without touching a 1.6k LOC file.

## Phase 9: Provider File Splits

Primary targets:
- `internal/nodes/cloud/aws/s3_operations.go`
- `internal/nodes/cloud/gcp/cloud_storage.go`
- `internal/nodes/cloud/azure/blob_storage.go`
- large node/provider files in `internal/nodes/*`

Problem:
- Some provider packages are already domain-isolated, but their files are too large and mix config, parsing, client bootstrapping, and operations.

Pattern:
- `types.go`
- `config.go`
- `client.go`
- `operations_*.go`
- `validation.go`
- `expressions.go`

Todo:
- [ ] Split large provider files without over-packaging them.
- [ ] Group operations by domain, such as object vs bucket operations.
- [ ] Keep provider-specific helpers local to their package.

Reuse focus:
- Shared parsing and validation within a provider package should be reusable across operations.

Acceptance:
- Provider packages stay cohesive while individual files become readable.

## Phase 10: Frontend Feature Decomposition

Primary targets:
- `web/src/components/copilot/AgentCopilot.vue`
- `web/src/composables/useKeyboardShortcuts.ts`

Target structure:
- `web/src/components/copilot/AgentCopilot.vue`
  shell and layout
- `web/src/components/copilot/CopilotChatTab.vue`
- `web/src/components/copilot/CopilotGenerateTab.vue`
- `web/src/components/copilot/CopilotSuggestTab.vue`
- `web/src/components/copilot/CopilotExplainTab.vue`
- `web/src/composables/useCopilotApi.ts`
- `web/src/composables/useKeyboardShortcuts.ts`
  generic registry only
- `web/src/composables/useWorkflowShortcuts.ts`
  editor-specific bindings

Todo:
- [ ] Split copilot tabs and API transport logic.
- [ ] Split generic shortcut registry from workflow-specific preset bindings.
- [ ] Keep tab components stateless where possible.

Reuse focus:
- Copilot transport should be reusable from multiple surfaces.
- Shortcut presets should be swappable per feature area.

Acceptance:
- Large feature components become shells over reusable tab/composable units.

## Cross-Cutting Reuse Rules

- Shared workflow manipulation belongs in a pure graph module, not in API handlers or Vue components.
- Shared transport helpers belong near transport boundaries, not inside domain modules.
- Shared provider parsing belongs inside provider packages unless it is genuinely cross-provider.
- Shared UI state belongs in composables/stores, not copied across views.
- Shared validation belongs in pure helpers with tests.

## Implementation Order

1. Phase 0: safety rails
2. Phase 1: backup artifact cleanup
3. Phase 2: API server decomposition
4. Phase 3: workflow graph core
5. Phase 4: JavaScript runtime split
6. Phase 5: scheduler modularization
7. Phase 6: marketplace/template split
8. Phase 7: CLI boundaries
9. Phase 8: expression library split
10. Phase 9: provider file splits
11. Phase 10: remaining frontend feature decomposition

## Working Rules During Execution

- Move code first, then rename exported symbols only if needed.
- Keep diffs narrow: one subsystem split per PR.
- Preserve existing tests where possible; add tests around extracted pure helpers.
- Avoid behavior changes during structural moves unless fixing a discovered correctness bug.
- When a bug is fixed during extraction, call it out explicitly in the change summary.

## First Execution Batch

Recommended first batch:
- [ ] Add refactor guardrails and capture baseline test commands
- [ ] Delete backup and disabled source artifacts
- [ ] Split `internal/api/server.go`
- [ ] Extract `web/src/utils/workflowGraph.ts`
- [ ] Split editor state from persistence state

This batch gives the highest clarity gain with the least architectural risk.
