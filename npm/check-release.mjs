// SPDX-License-Identifier: GPL-3.0-only
import { readFileSync } from 'node:fs';
import { release, validateRelease } from './lib/release.mjs';

const pkg = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(pkg.version)) throw new Error('Invalid npm launcher version');
validateRelease(release);
console.log('Validated six pinned binaries for ' + release.tag);
