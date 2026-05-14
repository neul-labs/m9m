/**
 * Binary download and caching for the m9m Node.js SDK.
 */

import * as fs from 'fs';
import * as https from 'https';
import * as os from 'os';
import * as path from 'path';

const GITHUB_REPO = 'neul-labs/m9m';

function getCacheDir(): string {
  const home = os.homedir();
  return path.join(home, '.m9m', 'bin');
}

function getPlatform(): { os: string; arch: string } {
  const platform = os.platform();
  const arch = os.arch();

  const osName: Record<string, string> = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };

  const archName: Record<string, string> = {
    x64: 'amd64',
    arm64: 'arm64',
  };

  return {
    os: osName[platform] || platform,
    arch: archName[arch] || arch,
  };
}

async function latestReleaseVersion(): Promise<string> {
  return new Promise((resolve) => {
    const req = https.get(
      `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`,
      {
        headers: {
          'User-Agent': 'm9m-nodejs-sdk/1.0',
          Accept: 'application/vnd.github.v3+json',
        },
      },
      (res) => {
        let data = '';
        res.on('data', (chunk) => (data += chunk));
        res.on('end', () => {
          try {
            const json = JSON.parse(data);
            resolve(json.tag_name || 'v0.2.0');
          } catch {
            resolve('v0.2.0');
          }
        });
      }
    );
    req.on('error', () => resolve('v0.2.0'));
    req.setTimeout(10000, () => {
      req.destroy();
      resolve('v0.2.0');
    });
  });
}

function downloadFile(url: string, dest: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const req = https.get(url, { headers: { 'User-Agent': 'm9m-nodejs-sdk/1.0' } }, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        if (res.headers.location) {
          downloadFile(res.headers.location, dest).then(resolve).catch(reject);
          return;
        }
      }
      if (res.statusCode !== 200) {
        reject(new Error(`Download failed: ${res.statusCode}`));
        return;
      }
      res.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    });
    req.on('error', reject);
    req.setTimeout(60000, () => {
      req.destroy();
      reject(new Error('Download timeout'));
    });
  });
}

export async function downloadBinary(version?: string): Promise<string> {
  const { os: osName, arch } = getPlatform();
  const ver = version || (await latestReleaseVersion());
  const artifactName = `m9m-${osName}-${arch}`;
  const url = `https://github.com/${GITHUB_REPO}/releases/download/${ver}/${artifactName}`;

  const cacheDir = getCacheDir();
  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
  }

  const binaryName = osName === 'windows' ? 'm9m.exe' : 'm9m';
  const cached = path.join(cacheDir, binaryName);

  await downloadFile(url, cached);

  if (osName !== 'windows') {
    fs.chmodSync(cached, 0o755);
  }

  return cached;
}

export function getBinaryPath(): string {
  const cacheDir = getCacheDir();
  const { os: osName } = getPlatform();
  const binaryName = osName === 'windows' ? 'm9m.exe' : 'm9m';
  const cached = path.join(cacheDir, binaryName);

  if (fs.existsSync(cached)) {
    return cached;
  }

  throw new Error(
    'm9m binary not found. Run downloadBinary() first or ensure it is downloaded during install.'
  );
}
