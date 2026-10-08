// Exercise the warm, bundled-version launcher with all network access disabled.
import { release } from '../lib/release.mjs';
process.env.FLEETSH_RELEASE = release.tag;
process.env.FLEETSH_OFFLINE = '1';
globalThis.fetch = () => { throw new Error('Warm launcher attempted a network request'); };
process.argv = [process.execPath, 'fleetsh', 'docs', 'forwarding'];
await import('../bin/fleetsh.mjs');
