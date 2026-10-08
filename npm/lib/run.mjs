// SPDX-License-Identifier: GPL-3.0-only
import { spawn } from 'node:child_process';
import { constants } from 'node:os';

export function runBinary(binary, args, { processRef = process, spawnImpl = spawn } = {}) {
  return new Promise((resolve, reject) => {
    const child = spawnImpl(binary, args, { stdio: 'inherit', shell: false, windowsHide: false });
    const handlers = new Map();
    for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
      const handler = () => { if (child.exitCode == null && child.signalCode == null) child.kill(signal); };
      handlers.set(signal, handler);
      processRef.on(signal, handler);
    }
    const cleanup = () => {
      for (const [signal, handler] of handlers) processRef.removeListener(signal, handler);
    };
    child.once('error', (error) => { cleanup(); reject(error); });
    child.once('close', (code, signal) => {
      cleanup();
      resolve(code ?? (128 + (constants.signals[signal] || 1)));
    });
  });
}
