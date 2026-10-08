// SPDX-License-Identifier: GPL-3.0-only
import * as fs from 'node:fs/promises';
import path from 'node:path';
import { randomUUID } from 'node:crypto';
import { cacheRoot, sha256 } from './cache.mjs';
import { download } from './download.mjs';
import { release, validateRelease, assetUrl } from './release.mjs';

const API = 'https://api.github.com/repos/KanataLabs/fleetsh/releases';
const TTL = 60 * 60 * 1000;
const validTag = (tag) => /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(tag || '');

async function jsonDownload(url, downloader, options = {}) {
  const bytes = await downloader(url, { maxSize: 4 * 1024 * 1024, timeoutMs: 15000, ...options });
  return JSON.parse(bytes.toString('utf8'));
}

function checkAssets(manifest, info) {
  validateRelease(manifest);
  if (manifest.tag !== info.tag_name || !Array.isArray(info.assets)) throw new Error('Release manifest tag mismatch');
  for (const asset of Object.values(manifest.assets)) {
    const published = info.assets.find((entry) => entry.name === asset.name);
    if (!published || published.state !== 'uploaded' || published.size !== asset.size ||
        published.digest !== 'sha256:' + asset.sha256) throw new Error('Release asset mismatch: ' + asset.name);
  }
}

async function manifestFor(info, downloader) {
  if (!info || typeof info !== 'object' || info.draft || !validTag(info.tag_name)) throw new Error('Invalid published release');
  if (!Array.isArray(info.assets) || info.assets.some((entry) => !entry || typeof entry !== 'object')) {
    throw new Error('Invalid release assets');
  }
  const source = info.assets.find((entry) => entry.name === 'fleetsh-manifest.json');
  if (!source) {
    // Compatibility with the initial release made before dynamic launchers existed.
    if (info.tag_name !== release.tag) throw new Error('Release is missing fleetsh-manifest.json');
    checkAssets(release, info);
    return release;
  }
  if (source.state !== 'uploaded' || !Number.isSafeInteger(source.size) ||
      source.size <= 0 || source.size > 65536 || !/^sha256:[a-f0-9]{64}$/.test(source.digest || '')) {
    throw new Error('Invalid release manifest asset');
  }
  const bytes = await downloader(assetUrl({ tag: info.tag_name }, source), { expectedSize: source.size, timeoutMs: 15000 });
  if ('sha256:' + sha256(bytes) !== source.digest) throw new Error('Release manifest checksum mismatch');
  const manifest = JSON.parse(bytes.toString('utf8'));
  checkAssets(manifest, info);
  return manifest;
}

async function newestPublished(downloader) {
  const releases = await jsonDownload(API + '?per_page=100', downloader);
  if (!Array.isArray(releases) || releases.some((info) => !info || typeof info !== 'object')) {
    throw new Error('Invalid release list');
  }
  const published = releases.filter((info) => !info.draft && validTag(info.tag_name) &&
    Number.isFinite(Date.parse(info.published_at)));
  published.sort((a, b) => Date.parse(b.published_at) - Date.parse(a.published_at));
  if (!published.length) throw new Error('No published fleetsh release is available');
  return published[0];
}

async function readMetadata(file, selector, now) {
  try {
    const stat = await fs.lstat(file);
    if (!stat.isFile() || stat.size > 65536) return undefined;
    const cached = JSON.parse(await fs.readFile(file, 'utf8'));
    validateRelease(cached.manifest);
    if (!Number.isFinite(cached.checkedAt) || cached.checkedAt > now ||
        (validTag(selector) && cached.manifest.tag !== selector)) return undefined;
    return cached;
  } catch (error) {
    if (error.code === 'ENOENT' || error instanceof SyntaxError || !error.code) return undefined;
    throw error;
  }
}

export async function resolveRelease({
  selector = 'latest', offline = false, refresh = false, root = cacheRoot(),
  downloader = download, now = Date.now(), log = (message) => process.stderr.write(message + '\n')
} = {}) {
  if (selector === 'bundled' || selector === release.tag) return release;
  if (!['latest', 'preview'].includes(selector) && !validTag(selector)) throw new Error('Invalid release selector');
  const file = path.join(root, 'metadata', selector + '.json');
  const cached = await readMetadata(file, selector, now);
  if (offline) {
    if (cached) return cached.manifest;
    if (selector === 'latest' || selector === 'preview') return release;
    throw new Error('No cached release metadata for ' + selector);
  }
  if (!refresh && cached && (validTag(selector) || now - cached.checkedAt < TTL)) return cached.manifest;
  let manifest;
  try {
    let info;
    if (validTag(selector)) info = await jsonDownload(API + '/tags/' + encodeURIComponent(selector), downloader);
    else if (selector === 'preview') info = await newestPublished(downloader);
    else {
      try {
        info = await jsonDownload(API + '/latest', downloader);
        if (info?.prerelease) throw new Error('Stable release endpoint returned a preview');
      } catch (error) {
        if (error.status !== 404) throw error;
        info = await newestPublished(downloader);
      }
    }
    if (validTag(selector) && info?.tag_name !== selector) throw new Error('Selected release tag mismatch');
    manifest = await manifestFor(info, downloader);
  } catch (error) {
    // Only availability failures may fall back; malformed releases and digests fail closed.
    const unavailable = error.name === 'TimeoutError' || error.name === 'TypeError' ||
      [403, 404, 408, 429].includes(error.status) || error.status >= 500;
    if (!unavailable) throw error;
    if (cached) {
      log('fleetsh: GitHub unavailable; using cached ' + cached.manifest.tag);
      return cached.manifest;
    }
    if (selector === 'latest' || selector === 'preview') {
      log('fleetsh: GitHub unavailable; using bundled ' + release.tag);
      return release;
    }
    throw error;
  }
  const directory = path.dirname(file);
  await fs.mkdir(directory, { recursive: true, mode: 0o700 });
  const temporary = path.join(directory, '.' + selector + '-' + randomUUID() + '.tmp');
  try {
    await fs.writeFile(temporary, JSON.stringify({ checkedAt: now, manifest }), { mode: 0o600, flag: 'wx' });
    await fs.rename(temporary, file);
  } finally {
    await fs.rm(temporary, { force: true });
  }
  return manifest;
}
