#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-only
import { ensureBinary } from '../lib/cache.mjs';
import { release } from '../lib/release.mjs';
import { runBinary } from '../lib/run.mjs';

try {
  const binary = await ensureBinary(release);
  process.exitCode = await runBinary(binary, process.argv.slice(2));
} catch (error) {
  process.stderr.write('fleetsh npm launcher: ' + error.message + '\n');
  process.exitCode = 2;
}
