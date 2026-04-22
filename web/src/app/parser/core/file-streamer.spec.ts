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

import { KHIFileStreamer } from 'src/app/parser/core/file-streamer';
import {
  ParserBlueprint,
  ChunkDefinition,
  IDataAssembler,
} from 'src/app/parser/core/interfaces';
import { InspectionData } from 'src/app/store/inspection-data';

describe('KHIFileStreamer', () => {
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

  let mockAssembler1: jasmine.SpyObj<IDataAssembler<string>>;
  let mockAssembler2: jasmine.SpyObj<IDataAssembler<number>>;
  let registry: Record<number, ParserBlueprint>;

  beforeEach(() => {
    mockAssembler1 = jasmine.createSpyObj('IDataAssembler', [
      'ingest',
      'assembleInto',
    ]);
    mockAssembler2 = jasmine.createSpyObj('IDataAssembler', [
      'ingest',
      'assembleInto',
    ]);

    const blueprint: ParserBlueprint = new Map<number, ChunkDefinition<any>>([
      [
        1,
        {
          typeId: 1,
          decode: (bytes) => new TextDecoder().decode(bytes),
          createAssembler: () => mockAssembler1,
          priority: 10,
        } as ChunkDefinition<string>,
      ],
      [
        2,
        {
          typeId: 2,
          decode: (bytes) => bytes[0],
          createAssembler: () => mockAssembler2,
          priority: 5,
        } as ChunkDefinition<number>,
      ],
    ]);

    registry = {
      6: blueprint,
    };
  });

  it('should throw an error for invalid magic bytes', async () => {
    const streamer = new KHIFileStreamer(registry);
    const buffer = new Uint8Array([88, 88, 88, 6]).buffer; // XXX\x06

    await expectAsync(streamer.parse(buffer)).toBeRejectedWithError(
      'Invalid magic bytes. Not a KHI file.',
    );
  });

  it('should throw an error for unsupported version', async () => {
    const streamer = new KHIFileStreamer(registry);
    const buffer = new Uint8Array([75, 72, 73, 99]).buffer; // KHI\x99

    await expectAsync(streamer.parse(buffer)).toBeRejectedWithError(
      'Unsupported KHI file version: 99',
    );
  });

  it('should parse chunks and assemble based on priority', async () => {
    const streamer = new KHIFileStreamer(registry);

    const data1 = new TextEncoder().encode('hello');
    const compressed1 = await compressData(data1);

    const data2 = new Uint8Array([42]);
    const compressed2 = await compressData(data2);

    const bufferSize = 4 + 8 + compressed1.length + 8 + compressed2.length;
    const buffer = new ArrayBuffer(bufferSize);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    uint8View.set([75, 72, 73, 6], 0);

    dv.setUint32(4, compressed1.length, true);
    dv.setUint32(8, 1, true);
    uint8View.set(compressed1, 12);

    const offset2 = 12 + compressed1.length;
    dv.setUint32(offset2, compressed2.length, true);
    dv.setUint32(offset2 + 4, 2, true);
    uint8View.set(compressed2, offset2 + 8);

    const result = await streamer.parse(buffer);

    expect(result).toBeNull(); // Because assembly logic is deferred to the builder

    expect(mockAssembler1.ingest).toHaveBeenCalledWith('hello');
    expect(mockAssembler2.ingest).toHaveBeenCalledWith(42);
  });
});
