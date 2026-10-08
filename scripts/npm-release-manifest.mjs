// SPDX-License-Identifier: GPL-3.0-only
// Verify a release's checksum list and binaries before updating the one npm package.
import { readFile, writeFile } from 'node:fs/promises';
import { extractBinary } from '../npm/lib/archive.mjs';
import { download } from '../npm/lib/download.mjs';
import { sha256 } from '../npm/lib/cache.mjs';
import { targets, validateRelease, assetUrl } from '../npm/lib/release.mjs';

const tag = process.argv[2];
if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(tag || '')) {
  throw new Error('Usage: node scripts/npm-release-manifest.mjs vVERSION');
}
const base = 'https://github.com/KanataLabs/fleetsh/releases/download/' + tag + '/';
const sums = (await download(base + 'SHA256SUMS')).toString('utf8');
const expected = new Map();
for (const line of sums.trim().split(/\r?\n/)) {
  const match = /^([a-f0-9]{64})\s+\*?(\S+)$/.exec(line);
  if (!match || expected.has(match[2])) throw new Error('Invalid or duplicate release checksum');
  expected.set(match[2], match[1]);
}
const manifest = { tag, assets: {} };
for (const target of targets) {
  const name = 'fleetsh_' + tag + '_' + target.replace('-', '_') +
    (target.startsWith('windows-') ? '.zip' : '.tar.gz');
  if (!expected.has(name)) throw new Error('Missing release checksum: ' + name);
  const archive = await download(assetUrl(manifest, { name }));
  if (sha256(archive) !== expected.get(name)) throw new Error('Release checksum mismatch: ' + name);
  manifest.assets[target] = {
    name, size: archive.length, sha256: expected.get(name),
    binarySha256: sha256(extractBinary(archive, target.split('-')[0]))
  };
  console.log('Verified ' + name);
}
const packageFile = new URL('../package.json', import.meta.url);
const pkg = JSON.parse(await readFile(packageFile, 'utf8'));
pkg.version = tag.slice(1);
validateRelease(manifest, pkg.version);
await writeFile(new URL('../npm/release.json', import.meta.url), JSON.stringify(manifest, null, 2) + '\n');
await writeFile(packageFile, JSON.stringify(pkg, null, 2) + '\n');
