// SPDX-License-Identifier: GPL-3.0-only
import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { extractBinary } from '../npm/lib/archive.mjs';
import { sha256 } from '../npm/lib/cache.mjs';
import { targets, validateRelease } from '../npm/lib/release.mjs';

const [tag, directory = 'dist'] = process.argv.slice(2);
if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(tag || '')) throw new Error('Expected vVERSION [directory]');
const manifest = { tag, assets: {} };
for (const target of targets) {
  const name = 'fleetsh_' + tag + '_' + target.replace('-', '_') + (target.startsWith('windows-') ? '.zip' : '.tar.gz');
  const archive = await readFile(path.join(directory, name));
  manifest.assets[target] = {
    name, size: archive.length, sha256: sha256(archive),
    binarySha256: sha256(extractBinary(archive, target.split('-')[0]))
  };
}
validateRelease(manifest);
await writeFile(path.join(directory, 'fleetsh-manifest.json'), JSON.stringify(manifest, null, 2) + '\n');
console.log('Generated verified release manifest for ' + tag);
