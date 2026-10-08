import assert from 'node:assert/strict';
import { test } from 'node:test';
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';
import { release } from '../lib/release.mjs';
import { resolveRelease } from '../lib/resolve.mjs';
import { parseOptions } from '../lib/options.mjs';
import { ensureBinary, sha256 } from '../lib/cache.mjs';
import { tarArchive, zipArchive } from './fixtures.mjs';

const API = 'https://api.github.com/repos/KanataLabs/fleetsh/releases';
const encode = (value) => Buffer.from(JSON.stringify(value));
const unavailable = (status) => Object.assign(new Error('HTTP ' + status), { status });

function future(tag = 'v0.2.0') {
  const manifest = structuredClone(release);
  manifest.tag = tag;
  for (const asset of Object.values(manifest.assets)) asset.name = asset.name.replace(release.tag, tag);
  return manifest;
}

function infoFor(manifest, { legacy = false, prerelease = false, published_at = '2026-10-08T00:00:00Z' } = {}) {
  const assets = Object.values(manifest.assets).map((asset) => ({
    name: asset.name, size: asset.size, state: 'uploaded', digest: 'sha256:' + asset.sha256
  }));
  if (!legacy) assets.push({ name: 'fleetsh-manifest.json', state: 'uploaded',
    size: encode(manifest).length, digest: 'sha256:' + sha256(encode(manifest)) });
  return { tag_name: manifest.tag, draft: false, prerelease, published_at, assets };
}

async function fixture(t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'fleetsh-npm-resolve-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  return root;
}

test('CLI selectors support original bundled binary, exact tags and channel choices', () => {
  assert.deepEqual(parseOptions(['--release', 'bundled', '--offline', 'stats', '@web'], {}),
    { selector: 'bundled', offline: true, refresh: false, help: false, args: ['stats', '@web'] });
  assert.equal(parseOptions(['--release=v0.2.0', 'stats'], { FLEETSH_RELEASE: 'preview' }).selector, 'v0.2.0');
  assert.equal(parseOptions(['--refresh', 'version'], {}).refresh, true);
  assert.equal(parseOptions(['--launcher-help'], {}).help, true);
  assert.deepEqual(parseOptions(['exec', 'hk1', '--release'], {}).args, ['exec', 'hk1', '--release']);
  assert.throws(() => parseOptions(['--release'], {}), /expects/);
  assert.throws(() => parseOptions(['--release=..\/outside'], {}), /expects/);
});

test('default prefers stable; TTL reuses metadata and refresh discovers new Go version without npm update', async (t) => {
  const root = await fixture(t);
  let selected = future();
  let calls = 0;
  const downloader = async (url) => {
    calls++;
    return url === API + '/latest' ? encode(infoFor(selected)) : encode(selected);
  };
  const options = { root, downloader, now: 10000, log: () => {} };
  assert.equal((await resolveRelease(options)).tag, 'v0.2.0');
  assert.equal(calls, 2);
  selected = future('v0.2.1');
  assert.equal((await resolveRelease({ ...options, now: 11000 })).tag, 'v0.2.0');
  assert.equal(calls, 2);
  assert.equal((await resolveRelease({ ...options, now: 12000, refresh: true })).tag, 'v0.2.1');
  assert.equal(calls, 4);
  selected = future('v0.2.2');
  assert.equal((await resolveRelease({ ...options, now: 12000 + 3600001 })).tag, 'v0.2.2');
  assert.equal(calls, 6);
  assert.equal(release.tag, 'v0.1.0-alpha.1');
});

test('when stable is absent the newest published preview is selected; initial legacy release stays compatible', async (t) => {
  const root = await fixture(t);
  const original = infoFor(release, { legacy: true, prerelease: true });
  const draft = { ...original, draft: true, published_at: '2027-01-01T00:00:00Z' };
  const downloader = async (url) => {
    if (url === API + '/latest') throw unavailable(404);
    assert.equal(url, API + '?per_page=100');
    return encode([draft, { ...original, tag_name: 'weekly' }, original]);
  };
  assert.deepEqual(await resolveRelease({ root, downloader }), release);
});

test('preview includes prereleases even when a stable release exists', async (t) => {
  const root = await fixture(t);
  const preview = future('v0.3.0-alpha.1');
  const downloader = async (url) => url === API + '?per_page=100' ?
    encode([infoFor(future(), { published_at: '2026-10-01T00:00:00Z' }), infoFor(preview, { prerelease: true })]) :
    encode(preview);
  assert.equal((await resolveRelease({ root, downloader, selector: 'preview' })).tag, preview.tag);
});

test('explicit version remains pinned while launcher version is independent', async (t) => {
  const root = await fixture(t);
  const pinned = future();
  let calls = 0;
  const downloader = async (url) => {
    calls++;
    return url === API + '/tags/v0.2.0' ? encode(infoFor(pinned)) : encode(pinned);
  };
  assert.equal((await resolveRelease({ root, downloader, selector: pinned.tag, now: 1000 })).tag, pinned.tag);
  assert.equal((await resolveRelease({ root, downloader, selector: pinned.tag, now: 9999999 })).tag, pinned.tag);
  assert.equal(calls, 2);
  const never = () => assert.fail('Bundled selection must not query GitHub');
  assert.equal((await resolveRelease({ root, selector: 'bundled', downloader: never })).tag, release.tag);
});

test('offline disables metadata and binary downloads, including a cold cache', async (t) => {
  const root = await fixture(t);
  const never = () => assert.fail('Offline must not download');
  assert.deepEqual(await resolveRelease({ root, offline: true, downloader: never }), release);
  await assert.rejects(ensureBinary(release, { root, offline: true, downloader: never }), /No cached binary/);
  await assert.rejects(resolveRelease({ root, selector: 'v0.2.0', offline: true, downloader: never }), /No cached release metadata/);
  assert.deepEqual(await fs.readdir(root), []);
});

test('network failures use verified cache; unavailable explicit tags never silently downgrade', async (t) => {
  const root = await fixture(t);
  const current = future();
  const downloader = async (url) => url === API + '/latest' ? encode(infoFor(current)) : encode(current);
  await resolveRelease({ root, downloader, now: 1000 });
  const logs = [];
  assert.equal((await resolveRelease({ root, refresh: true, downloader: async () => { throw unavailable(429); },
    now: 2000, log: (line) => logs.push(line) })).tag, current.tag);
  assert.match(logs[0], /using cached/);
  assert.equal((await resolveRelease({ root, selector: 'preview', downloader: async () => { throw unavailable(403); },
    log: () => {} })).tag, release.tag);
  await assert.rejects(resolveRelease({ root, selector: 'v9.0.0', downloader: async () => { throw unavailable(404); } }), /HTTP 404/);
});

test('bad manifest checksums, archive digests and malformed API data fail instead of downgrade', async (t) => {
  const root = await fixture(t);
  const manifest = future();
  const info = infoFor(manifest);
  for (const mode of ['checksum', 'archive', 'malformed']) {
    const altered = structuredClone(info);
    if (mode === 'checksum') altered.assets.at(-1).digest = 'sha256:' + '0'.repeat(64);
    if (mode === 'archive') altered.assets[0].digest = 'sha256:' + '0'.repeat(64);
    if (mode === 'malformed') altered.assets = {};
    const downloader = async (url) => url === API + '/latest' ? encode(altered) : encode(manifest);
    await assert.rejects(resolveRelease({ root, downloader, refresh: true }),
      /checksum mismatch|asset mismatch|Invalid release assets/);
  }
  await assert.rejects(resolveRelease({ root, downloader: async () => encode(null) }), /Invalid published release/);
  assert.deepEqual(await fs.readdir(root), []);
});

test('release workflow manifest builder hashes all six actual local archives without changing npm version', async (t) => {
  const root = await fixture(t);
  const tag = 'v0.2.0';
  for (const [target, asset] of Object.entries(future(tag).assets)) {
    const archive = target.startsWith('windows-') ? zipArchive() : tarArchive();
    await fs.writeFile(path.join(root, asset.name), archive);
  }
  const script = fileURLToPath(new URL('../../scripts/build-release-manifest.mjs', import.meta.url));
  await promisify(execFile)(process.execPath, [script, tag, root]);
  const manifest = JSON.parse(await fs.readFile(path.join(root, 'fleetsh-manifest.json'), 'utf8'));
  assert.equal(manifest.tag, tag);
  assert.equal(Object.keys(manifest.assets).length, 6);
  for (const asset of Object.values(manifest.assets)) {
    assert.equal(asset.binarySha256, sha256(Buffer.from('fixture-binary')));
    assert.equal(asset.sha256, sha256(await fs.readFile(path.join(root, asset.name))));
  }
});
