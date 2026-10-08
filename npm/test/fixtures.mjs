import { gzipSync, deflateRawSync } from 'node:zlib';

export function tarArchive(bytes = Buffer.from('fixture-binary'), type = '0', name = './fleetsh') {
  const header = Buffer.alloc(512);
  header.write(name);
  header.write('0000700\0', 100);
  header.write('0000000\0', 108);
  header.write('0000000\0', 116);
  header.write(bytes.length.toString(8).padStart(11, '0') + '\0', 124);
  header.write('00000000000\0', 136);
  header.fill(32, 148, 156);
  header.write(type, 156);
  header.write('ustar\0', 257);
  header.write('00', 263);
  const sum = header.reduce((a, b) => a + b, 0);
  header.write(sum.toString(8).padStart(6, '0') + '\0 ', 148);
  return gzipSync(Buffer.concat([header, bytes, Buffer.alloc((512 - bytes.length % 512) % 512 + 1024)]));
}

export function zipArchive(bytes = Buffer.from('fixture-binary'), { method = 8, descriptor = true, name = 'fleetsh.exe' } = {}) {
  const encoded = Buffer.from(name);
  const compressed = method === 0 ? bytes : deflateRawSync(bytes);
  const local = Buffer.alloc(30);
  local.writeUInt32LE(0x04034b50);
  local.writeUInt16LE(20, 4);
  local.writeUInt16LE(descriptor ? 8 : 0, 6);
  local.writeUInt16LE(method, 8);
  local.writeUInt32LE(descriptor ? 0 : compressed.length, 18);
  local.writeUInt32LE(descriptor ? 0 : bytes.length, 22);
  local.writeUInt16LE(encoded.length, 26);
  const trailing = descriptor ? Buffer.alloc(16) : Buffer.alloc(0);
  if (descriptor) {
    trailing.writeUInt32LE(0x08074b50);
    trailing.writeUInt32LE(compressed.length, 8);
    trailing.writeUInt32LE(bytes.length, 12);
  }
  const body = Buffer.concat([local, encoded, compressed, trailing]);
  const central = Buffer.alloc(46);
  central.writeUInt32LE(0x02014b50);
  central.writeUInt16LE(descriptor ? 8 : 0, 8);
  central.writeUInt16LE(method, 10);
  central.writeUInt32LE(compressed.length, 20);
  central.writeUInt32LE(bytes.length, 24);
  central.writeUInt16LE(encoded.length, 28);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50);
  end.writeUInt16LE(1, 8);
  end.writeUInt16LE(1, 10);
  end.writeUInt32LE(central.length + encoded.length, 12);
  end.writeUInt32LE(body.length, 16);
  return Buffer.concat([body, central, encoded, end]);
}
