// SPDX-License-Identifier: GPL-3.0-only
import { readFileSync } from 'node:fs';
import { release, validateRelease } from './lib/release.mjs';

const pkg = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
validateRelease(release, pkg.version);
console.log('Validated six pinned binaries for ' + release.tag);
