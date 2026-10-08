// SPDX-License-Identifier: GPL-3.0-only
import { createHash } from 'node:crypto';
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { extractBinary } from './archive.mjs';
import { download } from './download.mjs';
import { assetUrl, targetFor } from './release.mjs';

export const sha256 = (buffer) => createHash('sha256').update(buffer).digest('hex');

export function cacheRoot(env = process.env, platform = process.platform, home = os.homedir()) {
  if (env.FLEETSH_NPM_CACHE) return path.resolve(env.FLEETSH_NPM_CACHE);
  if (platform === 'win32') return path.join(env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'fleetsh', 'Cache', 'npm');
  if (platform === 'darwin') return path.join(home, 'Library', 'Caches', 'fleetsh', 'npm');
  return path.join(env.XDG_CACHE_HOME || path.join(home, '.cache'), 'fleetsh', 'npm');
}

async function readCached(binary, asset) {
  try {
    const stat = await fs.lstat(binary);
    if (!stat.isFile() || stat.size > 64 * 1024 * 1024 ||
        sha256(await fs.readFile(binary)) !== asset.binarySha256) {
      throw new Error('Cached binary checksum mismatch; remove ' + path.dirname(binary) + ' and retry.');
    }
    return binary;
  } catch (error) {
    if (error.code === 'ENOENT') return undefined;
    throw error;
  }
}

export async function ensureBinary(manifest, {
  platform = process.platform, arch = process.arch, root = cacheRoot(), offline = false,
  downloader = download, log = (message) => process.stderr.write(message + '\n')
} = {}) {
  const target = targetFor(platform, arch);
  const asset = manifest.assets[target];
  if (!asset || !/^[a-f0-9]{64}$/.test(asset.sha256) ||
      !/^[a-f0-9]{64}$/.test(asset.binarySha256) || !/^v[0-9A-Za-z.-]+$/.test(manifest.tag)) {
    throw new Error('Invalid release manifest');
  }
  const directory = path.join(root, manifest.tag, target, asset.sha256);
  const name = platform === 'win32' ? 'fleetsh.exe' : 'fleetsh';
  const binary = path.join(directory, name);
  const cached = await readCached(binary, asset);
  if (cached) return cached;
  if (offline) throw new Error('No cached binary for ' + manifest.tag + '; run once online before --offline.');
  const parent = path.dirname(directory);
  await fs.mkdir(parent, { recursive: true, mode: 0o700 });
  const staging = await fs.mkdtemp(path.join(parent, '.install-'));
  try {
    log('fleetsh: downloading ' + manifest.tag + ' for ' + target + ' (first run)');
    const archive = await downloader(assetUrl(manifest, asset), { expectedSize: asset.size });
    if (sha256(archive) !== asset.sha256) throw new Error('Release archive checksum mismatch');
    const bytes = extractBinary(archive, target.split('-')[0]);
    if (sha256(bytes) !== asset.binarySha256) throw new Error('Release binary checksum mismatch');
    await fs.writeFile(path.join(staging, name), bytes, { mode: 0o700, flag: 'wx' });
    try {
      await fs.rename(staging, directory);
    } catch (error) {
      // A concurrent first invocation may have atomically installed the same binary.
      if (!['EEXIST', 'ENOTEMPTY', 'EPERM'].includes(error.code) || !await readCached(binary, asset)) throw error;
    }
    return binary;
  } finally {
    await fs.rm(staging, { recursive: true, force: true });
  }
}
