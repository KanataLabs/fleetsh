// Exercise the warm launcher without allowing a release download.
globalThis.fetch = () => { throw new Error('Warm launcher attempted a network request'); };
process.argv = [process.execPath, 'fleetsh', 'docs', 'forwarding'];
await import('../bin/fleetsh.mjs');
