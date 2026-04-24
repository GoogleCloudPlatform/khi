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

import { assertNecessaryAPI } from 'src/app/common/misc-util';

/**
 * Data associated with a single chunk in the KHI file.
 */
export interface ChunkData {
  readonly size: number;
  readonly typeId: number;
  readonly data: Uint8Array;
}

/**
 * A utility class to sequentially read binary chunks from a KHI file ArrayBuffer.
 */
export class BinaryReader {
  private readonly dv: DataView;
  private offset = 0;

  /**
   * Creates a new BinaryReader for the given buffer.
   */
  constructor(private readonly buffer: ArrayBuffer) {
    assertNecessaryAPI('DecompressionStream');
    this.dv = new DataView(buffer);
  }

  /**
   * Returns the current byte offset being read.
   */
  public get currentOffset(): number {
    return this.offset;
  }

  /**
   * Checks if there is more data left in the buffer to read.
   */
  public hasMore(): boolean {
    return this.offset < this.buffer.byteLength;
  }

  /**
   * Reads and validates the header of the KHI file.
   */
  public readHeader(): { magic: string; version: number } {
    if (this.offset !== 0) {
      throw new Error('Offset must be 0 to read header');
    }
    if (this.buffer.byteLength < 4) {
      throw new Error('Buffer too small to contain header');
    }
    const magic = String.fromCharCode(
      this.dv.getUint8(0),
      this.dv.getUint8(1),
      this.dv.getUint8(2),
    );
    const version = this.dv.getUint8(3);
    this.offset = 4;
    return { magic, version };
  }

  /**
   * Reads the next chunk metadata and its binary payload, decompressing the payload using gzip.
   */
  public async readNextChunk(): Promise<ChunkData> {
    if (this.offset === 0) {
      throw new Error('Header must be read before reading chunks');
    }

    // 8 bytes chunk header (4 bytes size + 4 bytes typeId)
    if (this.offset + 8 > this.buffer.byteLength) {
      throw new Error('Unexpected end of file while reading chunk header');
    }

    // Chunk size (32 bit unsigned int, little endian)
    const size = this.dv.getUint32(this.offset, true);
    // Chunk type (32 bit unsigned int, little endian)
    const typeId = this.dv.getUint32(this.offset + 4, true);

    if (this.offset + 8 + size > this.buffer.byteLength) {
      throw new Error('Unexpected end of file while reading chunk data');
    }

    const compressedData = new Uint8Array(this.buffer, this.offset + 8, size);

    // Decompress the gzip data
    const decompressionStream = new DecompressionStream('gzip');
    const sourceBlob = new Blob([compressedData]);
    const textDecompressionStream = sourceBlob
      .stream()
      .pipeThrough(decompressionStream);
    const uncompressedBuffer = await new Response(
      textDecompressionStream,
    ).arrayBuffer();

    const data = new Uint8Array(uncompressedBuffer);

    this.offset += 8 + size;
    return { size, typeId, data };
  }
}
