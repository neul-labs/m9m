/**
 * Postinstall script — downloads the m9m binary after npm install.
 */

import { downloadBinary } from './binary';

async function main(): Promise<void> {
  // Skip in CI environments unless explicitly requested
  if (process.env.CI && !process.env.M9M_DOWNLOAD_BINARY) {
    console.log('[m9m] Skipping binary download in CI. Set M9M_DOWNLOAD_BINARY=1 to override.');
    return;
  }

  try {
    const path = await downloadBinary();
    console.log(`[m9m] Binary downloaded to ${path}`);
  } catch (err) {
    console.warn('[m9m] Binary download failed. You can manually download it later.');
    console.warn(String(err));
  }
}

main();
