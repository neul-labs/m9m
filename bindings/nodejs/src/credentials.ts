/**
 * CredentialManager stub for the m9m Node.js SDK.
 *
 * Full credential management requires the m9m server. This class provides
 * a local-only interface for storing credentials in memory.
 */

import type { CredentialData } from './types';

export class CredentialManager {
  private credentials: Map<string, CredentialData> = new Map();

  store(credential: CredentialData): boolean {
    this.credentials.set(credential.id, credential);
    return true;
  }

  storeMany(credentials: CredentialData[]): void {
    for (const cred of credentials) {
      this.store(cred);
    }
  }

  get(id: string): CredentialData | undefined {
    return this.credentials.get(id);
  }
}
