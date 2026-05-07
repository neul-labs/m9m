/**
 * m9m workflow engine Node.js SDK.
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
export { downloadBinary, getBinaryPath } from './binary';
export * from './types';

// Re-export version function
import { getBinaryPath } from './binary';
import { spawnSync } from 'child_process';

/**
 * Get the m9m library version.
 */
export function version(): string {
  try {
    const binary = getBinaryPath();
    const result = spawnSync(binary, ['version'], { encoding: 'utf8' });
    return result.stdout.trim() || 'unknown';
  } catch {
    return 'unknown';
  }
}
