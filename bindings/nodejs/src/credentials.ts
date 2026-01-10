/**
 * CredentialManager class for managing workflow credentials.
 */

import { getNativeBinding, NativeCredentialManager } from './native';
import type { CredentialData } from './types';

/**
 * Manages credentials for workflow nodes.
 *
 * @example
 * ```typescript
 * const credManager = new CredentialManager();
 * credManager.store({
 *   id: 'api-key-1',
 *   name: 'My API Key',
 *   type: 'apiKey',
 *   data: { apiKey: 'secret123' }
 * });
 *
 * const engine = new WorkflowEngine({ credentialManager: credManager });
 * ```
 */
export class CredentialManager {
  private native: NativeCredentialManager;

  /**
   * Create a new credential manager.
   */
  constructor() {
    const binding = getNativeBinding();
    this.native = new binding.CredentialManager();
  }

  /**
   * Store a credential.
   *
   * @param credential - The credential data to store.
   * @returns True if the credential was stored successfully.
   * @throws Error if storing fails.
   *
   * @example
   * ```typescript
   * credManager.store({
   *   id: 'slack-api',
   *   name: 'Slack API Token',
   *   type: 'slackApi',
   *   data: {
   *     accessToken: 'xoxb-...'
   *   }
   * });
   * ```
   */
  store(credential: CredentialData): boolean {
    return this.native.store(credential);
  }

  /**
   * Store multiple credentials.
   *
   * @param credentials - Array of credential data to store.
   * @throws Error if any credential fails to store.
   */
  storeMany(credentials: CredentialData[]): void {
    for (const cred of credentials) {
      this.store(cred);
    }
  }
}
