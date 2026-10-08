import assert from 'node:assert/strict';
import { test } from 'node:test';
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { gzipSync, gunzipSync } from 'node:zlib';
import { tarArchive, zipArchive } from './fixtures.mjs';
import { extractBinary } from '../lib/archive.mjs';
import { cacheRoot, ensureBinary, sha256 } from '../lib/cache.mjs';
import { download } from '../lib/download.mjs';
import { release, targetFor, validateRelease } from '../lib/release.mjs';

test('published manifest pins six complete archives and binaries', () => {
  const version = release.tag.slice(1);
  validateRelease(release, version);
  assert.throws(() => validateRelease(release, '0.0.0'), /must match/);
  assert.equal(targetFor('win32', 'x64'), 'windows-amd64');
  assert.equal(targetFor('darwin', 'arm64'), 'darwin-arm64');
  assert.throws(() => targetFor('linux', 'ia32'), /Unsupported platform/);
});

test('in-memory extraction supports tar, zip deflate, stored entries and descriptors', () => {
  const bytes = Buffer.from('native binary fixture');
  assert.deepEqual(extractBinary(tarArchive(bytes), 'linux'), bytes);
  for (const method of [0, 8]) for (const descriptor of [false, true]) {
    assert.deepEqual(extractBinary(zipArchive(bytes, { method, descriptor }), 'windows'), bytes);
  }
});

test('extraction refuses symlinks, missing binary paths and malformed archives', () => {
  assert.throws(() => extractBinary(tarArchive(undefined, '2'), 'linux'), /regular file/);
  assert.throws(() => extractBinary(tarArchive(undefined, '0', '../fleetsh'), 'linux'), /does not contain/);
  assert.throws(() => extractBinary(zipArchive(undefined, { name: '../fleetsh.exe' }), 'windows'), /does not contain/);
  const broken = gunzipSync(tarArchive());
  broken[0] ^= 1;
  assert.throws(() => extractBinary(gzipSync(broken), 'linux'), /checksum/);
  assert.throws(() => extractBinary(Buffer.from('broken'), 'windows'), /Invalid zip/);
  const truncated = zipArchive().subarray(0, -4);
  assert.throws(() => extractBinary(truncated, 'windows'), /Invalid zip/);
});

function fixtureManifest(archive, binary = Buffer.from('fixture-binary')) {
  return { tag: 'v0.1.0-alpha.1', assets: { 'linux-amd64': {
    name: 'fleetsh_v0.1.0-alpha.1_linux_amd64.tar.gz', size: archive.length,
    sha256: sha256(archive), binarySha256: sha256(binary)
  } } };
}

async function cacheFixture(t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'fleetsh-npm-test-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  return root;
}

test('first run verifies and installs; repeat run uses cache without network', async (t) => {
  const root = await cacheFixture(t);
  const archive = tarArchive();
  let downloads = 0;
  const options = { root, platform: 'linux', arch: 'x64', log: () => {},
    downloader: async (url, opts) => {
      downloads++;
      assert.match(url, /^https:\/\/github.com\/KanataLabs\/fleetsh\/releases\/download\/v0\.1\.0-alpha\.1\//);
      assert.equal(opts.expectedSize, archive.length);
      return archive;
    } };
  const manifest = fixtureManifest(archive);
  const binary = await ensureBinary(manifest, options);
  assert.deepEqual(await fs.readFile(binary), Buffer.from('fixture-binary'));
  assert.equal(await ensureBinary(manifest, options), binary);
  assert.equal(downloads, 1);
  await fs.writeFile(binary, 'corrupted');
  await assert.rejects(ensureBinary(manifest, options), /Cached binary checksum mismatch/);
  assert.equal(downloads, 1);
});

test('checksum failure never leaves an executable or temporary installation', async (t) => {
  const root = await cacheFixture(t);
  const archive = tarArchive();
  const manifest = fixtureManifest(archive);
  const options = { root, platform: 'linux', arch: 'x64', log: () => {}, downloader: async () => Buffer.from('tampered') };
  await assert.rejects(ensureBinary(manifest, options), /archive checksum mismatch/);
  const parent = path.join(root, manifest.tag, 'linux-amd64');
  assert.deepEqual(await fs.readdir(parent), []);
  const wrongBinary = fixtureManifest(archive, Buffer.from('different'));
  await assert.rejects(ensureBinary(wrongBinary, { ...options, downloader: async () => archive }), /binary checksum mismatch/);
  assert.deepEqual(await fs.readdir(parent), []);
});

test('concurrent cold starts atomically converge on one complete cache entry', async (t) => {
  const root = await cacheFixture(t);
  const archive = tarArchive();
  const manifest = fixtureManifest(archive);
  const options = { root, platform: 'linux', arch: 'x64', log: () => {},
    downloader: async () => { await new Promise((resolve) => setTimeout(resolve, 10)); return archive; } };
  const binaries = await Promise.all(Array.from({ length: 8 }, () => ensureBinary(manifest, options)));
  assert.equal(new Set(binaries).size, 1);
  assert.deepEqual(await fs.readFile(binaries[0]), Buffer.from('fixture-binary'));
  assert.deepEqual(await fs.readdir(path.dirname(path.dirname(binaries[0]))), [manifest.assets['linux-amd64'].sha256]);
});

test('unsupported target fails without network or cache writes', async (t) => {
  const root = await cacheFixture(t);
  await assert.rejects(ensureBinary(fixtureManifest(tarArchive()), { root, platform: 'freebsd',
    downloader: () => assert.fail('No download expected') }), /Unsupported platform/);
  assert.deepEqual(await fs.readdir(root), []);
});

test('cache paths stay separate from inventory and respect explicit override', () => {
  assert.equal(cacheRoot({ LOCALAPPDATA: 'C:/Local' }, 'win32', '/home'), path.join('C:/Local', 'fleetsh', 'Cache', 'npm'));
  assert.equal(cacheRoot({}, 'darwin', '/home'), path.join('/home', 'Library', 'Caches', 'fleetsh', 'npm'));
  assert.equal(cacheRoot({ XDG_CACHE_HOME: '/cache' }, 'linux', '/home'), path.join('/cache', 'fleetsh', 'npm'));
  assert.equal(cacheRoot({ FLEETSH_NPM_CACHE: './custom' }), path.resolve('./custom'));
});

test('download retries transient network failures once and never retries HTTP 404', async () => {
  let attempts = 0;
  assert.deepEqual(await download('https://example.test/archive', { expectedSize: 2, fetchImpl: async () => {
    if (++attempts === 1) throw new TypeError('fetch failed');
    return new Response('ok');
  } }), Buffer.from('ok'));
  assert.equal(attempts, 2);
  attempts = 0;
  await assert.rejects(download('https://example.test/archive', { fetchImpl: async () => {
    attempts++; return new Response(null, { status: 404 });
  } }), /HTTP 404/);
  assert.equal(attempts, 1);
});

test('download follows HTTPS redirects, bounds body size and rejects truncation', async () => {
  const requests = [];
  const fetchImpl = async (url, opts) => {
    requests.push(url);
    assert.equal(opts.redirect, 'manual');
    assert.ok(opts.signal instanceof AbortSignal);
    return requests.length === 1 ? new Response(null, { status: 302, headers: { location: 'https://assets.example.test/binary' } }) :
      new Response('bytes', { headers: { 'content-length': '5' } });
  };
  assert.deepEqual(await download('https://example.test/archive', { fetchImpl, expectedSize: 5 }), Buffer.from('bytes'));
  assert.equal(requests.length, 2);
  await assert.rejects(download('https://example.test/archive', { expectedSize: 3, fetchImpl: async () => new Response('bytes') }), /exceeds/);
  await assert.rejects(download('https://example.test/archive', { expectedSize: 6, fetchImpl: async () => new Response('bytes') }), /Truncated/);
  await assert.rejects(download('https://example.test/archive', { fetchImpl: async () =>
    new Response(null, { status: 302, headers: { location: 'http://example.test/archive' } }) }), /require HTTPS/);
  await assert.rejects(download('https://example.test/archive', { fetchImpl: async () => new Response(null, { status: 404 }) }), /HTTP 404/);
});
