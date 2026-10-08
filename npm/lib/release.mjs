// SPDX-License-Identifier: GPL-3.0-only
import { readFileSync } from 'node:fs';

export const release = JSON.parse(readFileSync(new URL('../release.json', import.meta.url), 'utf8'));
export const targets = ['darwin-amd64', 'darwin-arm64', 'linux-amd64', 'linux-arm64', 'windows-amd64', 'windows-arm64'];

export function validateRelease(manifest, version) {
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version) || manifest.tag !== 'v' + version) {
    throw new Error('npm version must match the pinned Go release');
  }
  if (Object.keys(manifest.assets).length !== targets.length) throw new Error('Expected six release targets');
  for (const target of targets) {
    const asset = manifest.assets[target];
    const extension = target.startsWith('windows-') ? '.zip' : '.tar.gz';
    const name = 'fleetsh_' + manifest.tag + '_' + target.replace('-', '_') + extension;
    if (!asset || asset.name !== name || !Number.isSafeInteger(asset.size) ||
        asset.size <= 0 || asset.size > 16 * 1024 * 1024 ||
        !/^[a-f0-9]{64}$/.test(asset.sha256) || !/^[a-f0-9]{64}$/.test(asset.binarySha256)) {
      throw new Error('Invalid pinned release asset: ' + target);
    }
  }
}

export function targetFor(platform = process.platform, arch = process.arch) {
  const os = { darwin: 'darwin', linux: 'linux', win32: 'windows' }[platform];
  const cpu = { x64: 'amd64', arm64: 'arm64' }[arch];
  if (!os || !cpu) throw new Error('Unsupported platform: ' + platform + '/' + arch +
    '. fleetsh supports Windows, macOS and Linux on x64/arm64.');
  return os + '-' + cpu;
}

export function assetUrl(manifest, asset) {
  return 'https://github.com/KanataLabs/fleetsh/releases/download/' +
    encodeURIComponent(manifest.tag) + '/' + encodeURIComponent(asset.name);
}
