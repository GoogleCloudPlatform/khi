/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { BinaryReader } from 'src/app/parser/core/binary-reader';

describe('BinaryReader', () => {
  /**
   * Helper to compress a Uint8Array using gzip
   */
  async function compressData(data: Uint8Array): Promise<Uint8Array> {
    const CompressionStreamAPI = globalThis.CompressionStream;
    if (!CompressionStreamAPI) {
      throw new Error(
        'CompressionStream API is not supported in this environment.',
      );
    }
    const compressionStream = new CompressionStreamAPI('gzip');
    const sourceBlob = new Blob([data as unknown as BlobPart]);
    const compressedStream = sourceBlob.stream().pipeThrough(compressionStream);
    const compressedBuffer = await new Response(compressedStream).arrayBuffer();
    return new Uint8Array(compressedBuffer);
  }

  it('should read the header correctly', () => {
    // Magic: K (75), H (72), I (73), Version: 6
    const buffer = new Uint8Array([75, 72, 73, 6]).buffer;
    const reader = new BinaryReader(buffer);

    const header = reader.readHeader();

    expect(header.magic).toBe('KHI');
    expect(header.version).toBe(6);
    expect(reader.currentOffset).toBe(4);
    expect(reader.hasMore()).toBeFalse();
  });

  it('should throw an error if the buffer is too small for a header', () => {
    const buffer = new Uint8Array([75, 72, 73]).buffer;
    const reader = new BinaryReader(buffer);

    expect(() => reader.readHeader()).toThrowError(
      'Buffer too small to contain header',
    );
  });

  it('should read chunks correctly', async () => {
    const data1 = new Uint8Array([10, 20]);
    const compressed1 = await compressData(data1);

    const data2 = new Uint8Array([30, 40, 50]);
    const compressed2 = await compressData(data2);

    // Compute total buffer size
    const bufferSize = 4 + 8 + compressed1.length + 8 + compressed2.length;
    const buffer = new ArrayBuffer(bufferSize);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    // Header
    uint8View.set([75, 72, 73, 6], 0);

    // Chunk 1 Header
    dv.setUint32(4, compressed1.length, true);
    dv.setUint32(8, 1, true); // type = 1
    uint8View.set(compressed1, 12);

    // Chunk 2 Header
    const offset2 = 12 + compressed1.length;
    dv.setUint32(offset2, compressed2.length, true);
    dv.setUint32(offset2 + 4, 2, true); // type = 2
    uint8View.set(compressed2, offset2 + 8);

    const reader = new BinaryReader(buffer);
    reader.readHeader();

    expect(reader.hasMore()).toBeTrue();

    const chunk1 = await reader.readNextChunk();
    expect(chunk1.size).toBe(compressed1.length);
    expect(chunk1.typeId).toBe(1);
    expect(Array.from(chunk1.data)).toEqual(Array.from(data1));
    expect(reader.currentOffset).toBe(offset2);

    expect(reader.hasMore()).toBeTrue();

    const chunk2 = await reader.readNextChunk();
    expect(chunk2.size).toBe(compressed2.length);
    expect(chunk2.typeId).toBe(2);
    expect(Array.from(chunk2.data)).toEqual(Array.from(data2));
    expect(reader.currentOffset).toBe(bufferSize);

    expect(reader.hasMore()).toBeFalse();
  });

  it('should throw an error if chunk header is incomplete', async () => {
    // Header (4) + incomplete chunk header (4)
    const buffer = new Uint8Array([75, 72, 73, 6, 2, 0, 0, 0]).buffer;
    const reader = new BinaryReader(buffer);
    reader.readHeader();

    await expectAsync(reader.readNextChunk()).toBeRejectedWithError(
      'Unexpected end of file while reading chunk header',
    );
  });

  it('should throw an error if chunk data is incomplete', async () => {
    // Header (4) + chunk header (size=10, type=1) + incomplete data (2)
    const buffer = new Uint8Array([
      75, 72, 73, 6, 10, 0, 0, 0, 1, 0, 0, 0, 255, 255,
    ]).buffer;
    const reader = new BinaryReader(buffer);
    reader.readHeader();

    await expectAsync(reader.readNextChunk()).toBeRejectedWithError(
      'Unexpected end of file while reading chunk data',
    );
  });
});
