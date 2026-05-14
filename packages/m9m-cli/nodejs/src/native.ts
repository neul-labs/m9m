/**
 * Native binding loader for the m9m library.
 */

import * as path from 'path';
import * as fs from 'fs';

interface NativeBinding {
  Engine: new (credentialManager?: unknown) => NativeEngine;
  Workflow: {
    fromJSON: (json: string | object) => NativeWorkflow;
    fromFile: (path: string) => NativeWorkflow;
  };
  CredentialManager: new () => NativeCredentialManager;
  version: () => string;
}

interface NativeEngine {
  execute: (workflow: NativeWorkflow, input?: unknown[]) => unknown;
  registerNode: (nodeType: string, callback: Function) => void;
}

interface NativeWorkflow {
  toJSON: () => unknown;
  name: string | null;
  id: string | null;
}

interface NativeCredentialManager {
  store: (credential: object) => boolean;
}

let nativeBinding: NativeBinding | null = null;

/**
 * Find the native addon.
 */
function findNativeAddon(): string | null {
  const possiblePaths = [
    // Built addon in build/Release
    path.join(__dirname, '..', 'build', 'Release', 'm9m.node'),
    // Built addon in build/Debug
    path.join(__dirname, '..', 'build', 'Debug', 'm9m.node'),
    // Pre-built addon in lib
    path.join(__dirname, '..', 'lib', 'm9m.node'),
  ];

  for (const addonPath of possiblePaths) {
    if (fs.existsSync(addonPath)) {
      return addonPath;
    }
  }

  return null;
}

/**
 * Get the native binding (lazy loading).
 */
export function getNativeBinding(): NativeBinding {
  if (nativeBinding) {
    return nativeBinding;
  }

  const addonPath = findNativeAddon();
  if (!addonPath) {
    throw new Error(
      'Native addon not found. Please run "npm run build:native" to build the addon, ' +
        'or ensure the m9m shared library is built and in the correct location.'
    );
  }

  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    nativeBinding = require(addonPath) as NativeBinding;
    return nativeBinding;
  } catch (err) {
    throw new Error(
      `Failed to load native addon from ${addonPath}: ${err instanceof Error ? err.message : String(err)}`
    );
  }
}

/**
 * Check if native bindings are available.
 */
export function isNativeAvailable(): boolean {
  try {
    getNativeBinding();
    return true;
  } catch {
    return false;
  }
}

export type { NativeBinding, NativeEngine, NativeWorkflow, NativeCredentialManager };
