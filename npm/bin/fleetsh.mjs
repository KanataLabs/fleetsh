#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-only
import { ensureBinary } from '../lib/cache.mjs';
import { parseOptions, launcherHelp } from '../lib/options.mjs';
import { resolveRelease } from '../lib/resolve.mjs';
import { targetFor } from '../lib/release.mjs';
import { runBinary } from '../lib/run.mjs';

try {
  const options = parseOptions(process.argv.slice(2));
  if (options.help) process.stdout.write(launcherHelp);
  else {
    if (options.args.length === 1 && ['--help', '-h'].includes(options.args[0])) process.stdout.write(launcherHelp + '\n');
    targetFor();
    const manifest = await resolveRelease(options);
    const binary = await ensureBinary(manifest, { offline: options.offline });
    process.exitCode = await runBinary(binary, options.args);
  }
} catch (error) {
  process.stderr.write('fleetsh npm launcher: ' + error.message + '\n');
  process.exitCode = 2;
}
