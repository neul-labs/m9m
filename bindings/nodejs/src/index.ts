/**
 * m9m workflow engine Node.js bindings.
 *
 * @example
 * ```typescript
 * import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';
 *
 * const engine = new WorkflowEngine();
 * const workflow = Workflow.fromJSON({ name: 'test', nodes: [], connections: {} });
 * const result = await engine.execute(workflow);
 * console.log(result.data);
 * ```
 */

export { WorkflowEngine } from './engine';
export { Workflow } from './workflow';
export { CredentialManager } from './credentials';
export * from './types';

// Re-export version function
import { getNativeBinding } from './native';

/**
 * Get the m9m library version.
 */
export function version(): string {
  try {
    const native = getNativeBinding();
    return native.version();
  } catch {
    return 'unknown';
  }
}
