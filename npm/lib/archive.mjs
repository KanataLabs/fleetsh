// SPDX-License-Identifier: GPL-3.0-only
import { gunzipSync, inflateRawSync } from 'node:zlib';

const MAX_BINARY = 64 * 1024 * 1024;
const text = (buffer) => buffer.toString('utf8').split('\0')[0];

export function extractBinary(archive, platform) {
  return platform === 'windows' ? extractZip(archive) : extractTar(archive);
}

function extractTar(archive) {
  const tar = gunzipSync(archive, { maxOutputLength: 128 * 1024 * 1024 });
  for (let offset = 0; offset + 512 <= tar.length;) {
    const header = tar.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) break;
    const octal = (start, length) => {
      const value = text(header.subarray(start, start + length)).trim();
      if (!/^[0-7]+$/.test(value)) throw new Error('Invalid tar header');
      return Number.parseInt(value, 8);
    };
    const checksum = header.reduce((sum, byte, index) =>
      sum + (index >= 148 && index < 156 ? 32 : byte), 0);
    if (checksum !== octal(148, 8)) throw new Error('Invalid tar checksum');
    const size = octal(124, 12);
    const end = offset + 512 + size;
    if (!Number.isSafeInteger(end) || end > tar.length) throw new Error('Truncated tar entry');
    const prefix = text(header.subarray(345, 500));
    const name = (prefix ? prefix + '/' : '') + text(header.subarray(0, 100));
    if (name === 'fleetsh' || name === './fleetsh') {
      if (header[156] !== 0 && header[156] !== 48) throw new Error('Binary is not a regular file');
      if (size === 0 || size > MAX_BINARY) throw new Error('Invalid binary size');
      return Buffer.from(tar.subarray(offset + 512, end));
    }
    offset += 512 + Math.ceil(size / 512) * 512;
  }
  throw new Error('Release archive does not contain fleetsh');
}

function extractZip(zip) {
  let eocd = -1;
  for (let i = zip.length - 22; i >= Math.max(0, zip.length - 65557); i--) {
    if (zip.readUInt32LE(i) === 0x06054b50 &&
        i + 22 + zip.readUInt16LE(i + 20) === zip.length) { eocd = i; break; }
  }
  if (eocd < 0) throw new Error('Invalid zip archive');
  if (zip.readUInt16LE(eocd + 4) || zip.readUInt16LE(eocd + 6)) {
    throw new Error('Split zip archives are not supported');
  }
  const count = zip.readUInt16LE(eocd + 10);
  const centralSize = zip.readUInt32LE(eocd + 12);
  const centralStart = zip.readUInt32LE(eocd + 16);
  if (centralStart + centralSize !== eocd) throw new Error('Invalid zip directory');
  let offset = centralStart;
  for (let index = 0; index < count; index++) {
    if (offset + 46 > eocd || zip.readUInt32LE(offset) !== 0x02014b50) {
      throw new Error('Invalid zip directory entry');
    }
    const flags = zip.readUInt16LE(offset + 8);
    const method = zip.readUInt16LE(offset + 10);
    const compressedSize = zip.readUInt32LE(offset + 20);
    const size = zip.readUInt32LE(offset + 24);
    const nameLength = zip.readUInt16LE(offset + 28);
    const extraLength = zip.readUInt16LE(offset + 30);
    const commentLength = zip.readUInt16LE(offset + 32);
    const local = zip.readUInt32LE(offset + 42);
    const next = offset + 46 + nameLength + extraLength + commentLength;
    if (next > eocd) throw new Error('Truncated zip directory');
    const name = zip.subarray(offset + 46, offset + 46 + nameLength).toString('utf8');
    if (name === 'fleetsh.exe') {
      if ((flags & 1) || size === 0 || size > MAX_BINARY || ![0, 8].includes(method)) {
        throw new Error('Unsupported zip binary entry');
      }
      if (local + 30 > centralStart || zip.readUInt32LE(local) !== 0x04034b50 ||
          zip.readUInt16LE(local + 8) !== method) throw new Error('Invalid zip local header');
      const localNameLength = zip.readUInt16LE(local + 26);
      const start = local + 30 + localNameLength + zip.readUInt16LE(local + 28);
      if (start + compressedSize > centralStart ||
          zip.subarray(local + 30, local + 30 + localNameLength).toString('utf8') !== name) {
        throw new Error('Truncated zip binary');
      }
      const compressed = zip.subarray(start, start + compressedSize);
      const binary = method === 0 ? Buffer.from(compressed) :
        inflateRawSync(compressed, { maxOutputLength: MAX_BINARY });
      if (binary.length !== size) throw new Error('Invalid zip binary size');
      return binary;
    }
    offset = next;
  }
  throw new Error('Release archive does not contain fleetsh.exe');
}
