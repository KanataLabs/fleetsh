import assert from 'node:assert/strict';
import { EventEmitter } from 'node:events';
import { spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { runBinary } from '../lib/run.mjs';

test('runner inherits terminal handles, forwards signals and cleans handlers', async () => {
  const child = new EventEmitter();
  child.killed = false;
  const killed = [];
  child.kill = (signal) => killed.push(signal);
  const processRef = new EventEmitter();
  const args = ['ssh', 'host name', '$(literal)', '; & spaces'];
  const pending = runBinary('/binary', args, { processRef, spawnImpl: (binary, actualArgs, options) => {
    assert.equal(binary, '/binary');
    assert.equal(actualArgs, args);
    assert.equal(options.stdio, 'inherit');
    assert.equal(options.shell, false);
    return child;
  } });
  processRef.emit('SIGINT');
  assert.deepEqual(killed, ['SIGINT']);
  child.emit('close', 3, null);
  assert.equal(await pending, 3);
  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) assert.equal(processRef.listenerCount(signal), 0);
});

test('runner reports spawn failures and signal exits', async () => {
  const child = new EventEmitter();
  child.kill = () => {};
  const processRef = new EventEmitter();
  const pending = runBinary('/missing', [], { processRef, spawnImpl: () => child });
  child.emit('error', new Error('ENOENT'));
  await assert.rejects(pending, /ENOENT/);
  assert.equal(processRef.listenerCount('SIGINT'), 0);
  const another = new EventEmitter();
  const signaled = runBinary('/binary', [], { processRef, spawnImpl: () => another });
  another.emit('close', null, 'SIGINT');
  assert.equal(await signaled, 130);
});

async function childFixture(t) {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'fleetsh-npm-run-'));
  t.after(() => fs.rm(directory, { recursive: true, force: true }));
  const runner = path.join(directory, 'runner.mjs');
  const child = path.join(directory, 'child.mjs');
  const moduleUrl = new URL('../lib/run.mjs', import.meta.url).href;
  await fs.writeFile(runner, 'import {runBinary} from ' + JSON.stringify(moduleUrl) +
    '; process.exitCode = await runBinary(process.execPath, JSON.parse(process.argv[2]));');
  await fs.writeFile(child, `let input = ''; for await (const chunk of process.stdin) input += chunk;
process.stdout.write(JSON.stringify({args: process.argv.slice(2), input, cwd: process.cwd()}));
process.stderr.write('child stderr');
process.exitCode = 3;`);
  return { runner, child, directory: await fs.realpath(directory) };
}

test('real child preserves stdin, literal arguments, working directory, stdout/stderr and exit code', async (t) => {
  const { runner, child, directory } = await childFixture(t);
  const literal = ['space value', '$HOME', '; exit 9', '$(echo nope)', '@web', '--json'];
  const result = await new Promise((resolve, reject) => {
    const process = spawn(globalThis.process.execPath, [runner, JSON.stringify([child, ...literal])], { cwd: directory });
    let stdout = '', stderr = '';
    process.stdout.on('data', (chunk) => stdout += chunk);
    process.stderr.on('data', (chunk) => stderr += chunk);
    process.on('error', reject);
    process.on('close', (code) => resolve({ code, stdout, stderr }));
    process.stdin.end('hidden input\n');
  });
  assert.equal(result.code, 3);
  assert.deepEqual(JSON.parse(result.stdout), { args: literal, input: 'hidden input\n', cwd: directory });
  assert.equal(result.stderr, 'child stderr');
});

test('real runner forwards SIGTERM to the child', { skip: process.platform === 'win32', timeout: 10000 }, async (t) => {
  const { runner, child } = await childFixture(t);
  await fs.writeFile(child, "process.on('SIGTERM', () => { console.log('terminated'); process.exit(0); }); console.log('ready'); setInterval(() => {}, 1000);");
  const result = await new Promise((resolve, reject) => {
    const wrapper = spawn(process.execPath, [runner, JSON.stringify([child])]);
    t.after(() => { if (wrapper.exitCode === null) wrapper.kill('SIGKILL'); });
    let stdout = '';
    let signaled = false;
    wrapper.stdout.on('data', (chunk) => {
      stdout += chunk;
      if (!signaled && stdout.includes('ready')) { signaled = true; wrapper.kill('SIGTERM'); }
    });
    wrapper.on('error', reject);
    wrapper.on('close', (code) => resolve({ code, stdout }));
  });
  assert.equal(result.code, 0);
  assert.match(result.stdout, /terminated/);
});
