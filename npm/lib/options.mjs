// SPDX-License-Identifier: GPL-3.0-only
export function parseOptions(argv, env = process.env) {
  const options = { args: [...argv], selector: env.FLEETSH_RELEASE || 'latest',
    offline: env.FLEETSH_OFFLINE === '1', refresh: env.FLEETSH_REFRESH === '1', help: false };
  while (options.args.length) {
    const first = options.args[0];
    if (first === '--release' || first.startsWith('--release=')) {
      options.args.shift();
      options.selector = first === '--release' ? options.args.shift() : first.slice(10);
    } else if (first === '--offline' || first === '--refresh' || first === '--launcher-help') {
      options.args.shift();
      options[first === '--offline' ? 'offline' : first === '--refresh' ? 'refresh' : 'help'] = true;
    } else break;
  }
  if (!['latest', 'preview', 'bundled'].includes(options.selector) &&
      !/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(options.selector || '')) {
    throw new Error('--release expects latest, preview, bundled or an exact vVERSION');
  }
  return options;
}

export const launcherHelp = `fleetsh npm launcher options (place before the Go command):
  --release latest     Latest stable release; preview only when no stable exists (default)
  --release preview    Most recently published release, including previews
  --release bundled    Original binary pinned in this launcher
  --release vVERSION   Exact Go release, independent of the npm package version
  --refresh            Check GitHub now instead of using the one-hour metadata cache
  --offline            Use cached binaries without GitHub requests
  --launcher-help      Print these options without downloading a binary

Examples:
  npx fleetsh stats
  npx fleetsh --release bundled version
  npx fleetsh --release v0.1.0-alpha.1 stats
  npx fleetsh --refresh version
  npx fleetsh --offline stats

Environment: FLEETSH_RELEASE, FLEETSH_REFRESH=1, FLEETSH_OFFLINE=1,
             FLEETSH_NPM_CACHE (optional cache directory).
`;
