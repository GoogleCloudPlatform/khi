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

import { KHIFileParser } from 'src/app/parser/core/file-parser';
import {
  KHIChunkDecodeError,
  KHIInvalidFileError,
} from 'src/app/parser/errors/parser-errors';
import {
  ParserBlueprint,
  ChunkDefinition,
  DataAssembler,
} from 'src/app/parser/core/interfaces';
import { ProgressReporter } from 'src/app/services/progress/progress-interface';

describe('KHIFileParser', () => {
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

  let mockAssembler1: jasmine.SpyObj<Required<DataAssembler<string>>>;
  let mockAssembler2: jasmine.SpyObj<Required<DataAssembler<number>>>;
  let registry: Record<number, ParserBlueprint>;

  beforeEach(() => {
    mockAssembler1 = jasmine.createSpyObj('DataAssembler', [
      'ingest',
      'finalize',
    ]);
    mockAssembler2 = jasmine.createSpyObj('DataAssembler', [
      'ingest',
      'finalize',
    ]);

    const blueprint: ParserBlueprint = new Map<
      number,
      ChunkDefinition<unknown>
    >([
      [
        1,
        {
          typeId: 1,
          label: 'string-chunk',
          decode: (bytes) => new TextDecoder().decode(bytes),
          createAssembler: () => mockAssembler1,
          priority: 10,
        } as ChunkDefinition<string>,
      ],
      [
        2,
        {
          typeId: 2,
          label: 'number-chunk',
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
    const parser = new KHIFileParser(registry);
    const buffer = new Uint8Array([88, 88, 88, 6]).buffer; // XXX\x06

    await expectAsync(parser.parse(buffer)).toBeRejectedWithError(
      'Invalid magic bytes. Not a KHI file.',
    );
  });

  it('should throw an error for unsupported version', async () => {
    const parser = new KHIFileParser(registry);
    const buffer = new Uint8Array([75, 72, 73, 99]).buffer; // KHI\x99

    await expectAsync(parser.parse(buffer)).toBeRejectedWithError(
      'Unsupported KHI file version: 99',
    );
  });

  it('should parse chunks and assemble based on priority', async () => {
    const parser = new KHIFileParser(registry);

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

    const result = await parser.parse(buffer);

    expect(result).toBeDefined();
    expect(result.internPool).toBeDefined();
    expect(result.styleStore).toBeDefined();
    expect(result.logStore).toBeDefined();
    expect(result.timelineStore).toBeDefined();

    expect(mockAssembler1.ingest).toHaveBeenCalledWith('hello');
    expect(mockAssembler2.ingest).toHaveBeenCalledWith(42);

    expect(mockAssembler1.finalize).toHaveBeenCalled();
    expect(mockAssembler2.finalize).toHaveBeenCalled();
    expect(mockAssembler2.finalize).toHaveBeenCalledBefore(
      mockAssembler1.finalize,
    );
  });

  it('should throw KHIDataAssemblyError if assembler fails during assembly', async () => {
    mockAssembler1.finalize.and.throwError('Assembly failed');
    const parser = new KHIFileParser(registry);

    const data1 = new TextEncoder().encode('hello');
    const compressed1 = await compressData(data1);

    const buffer = new ArrayBuffer(4 + 8 + compressed1.length);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    uint8View.set([75, 72, 73, 6], 0);
    dv.setUint32(4, compressed1.length, true);
    dv.setUint32(8, 1, true);
    uint8View.set(compressed1, 12);

    await expectAsync(parser.parse(buffer)).toBeRejectedWithError(
      /Failed to assemble data for chunk type 1 in version 6/,
    );
  });

  it('should accept progressReporter and parse chunks successfully', async () => {
    const parser = new KHIFileParser(registry);

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

    const mockProgressReporter: jasmine.SpyObj<ProgressReporter> =
      jasmine.createSpyObj('ProgressReporter', [
        'reportMessage',
        'reportProgress',
        'complete',
      ]);

    const result = await parser.parse(buffer, mockProgressReporter);

    expect(result).toBeDefined();
    expect(mockProgressReporter.reportMessage).toHaveBeenCalled();
    expect(mockProgressReporter.reportProgress).toHaveBeenCalledWith(0);
    expect(mockProgressReporter.reportProgress).toHaveBeenCalledWith(100);
    expect(mockProgressReporter.complete).toHaveBeenCalled();
  });

  it('should skip unhandled chunk types without decompressing or erroring', async () => {
    const parser = new KHIFileParser(registry);

    const data1 = new TextEncoder().encode('hello');
    const compressed1 = await compressData(data1);

    // Dummy compressed data for chunk type 99 (not in blueprint)
    const dataUnhandled = new Uint8Array([1, 2, 3, 4]);
    const compressedUnhandled = await compressData(dataUnhandled);

    const data2 = new Uint8Array([42]);
    const compressed2 = await compressData(data2);

    const bufferSize =
      4 +
      8 +
      compressed1.length +
      8 +
      compressedUnhandled.length +
      8 +
      compressed2.length;
    const buffer = new ArrayBuffer(bufferSize);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    uint8View.set([75, 72, 73, 6], 0);

    // Chunk 1: Type 1 (Handled)
    dv.setUint32(4, compressed1.length, true);
    dv.setUint32(8, 1, true);
    uint8View.set(compressed1, 12);

    // Chunk 2: Type 99 (Unhandled / Server-only, Should be skipped)
    const offset2 = 12 + compressed1.length;
    dv.setUint32(offset2, compressedUnhandled.length, true);
    dv.setUint32(offset2 + 4, 99, true);
    uint8View.set(compressedUnhandled, offset2 + 8);

    // Chunk 3: Type 2 (Handled)
    const offset3 = offset2 + 8 + compressedUnhandled.length;
    dv.setUint32(offset3, compressed2.length, true);
    dv.setUint32(offset3 + 4, 2, true);
    uint8View.set(compressed2, offset3 + 8);

    const result = await parser.parse(buffer);

    expect(result).toBeDefined();
    expect(mockAssembler1.ingest).toHaveBeenCalledWith('hello');
    expect(mockAssembler2.ingest).toHaveBeenCalledWith(42);
    expect(mockAssembler1.finalize).toHaveBeenCalled();
    expect(mockAssembler2.finalize).toHaveBeenCalled();
  });

  it('should accurately report chunkIndex in decode error when unhandled chunks are skipped', async () => {
    mockAssembler2.ingest.and.throwError('Decode failed');
    const parser = new KHIFileParser(registry);

    const dataUnhandled = new Uint8Array([1, 2, 3]);
    const compressedUnhandled = await compressData(dataUnhandled);

    const data2 = new Uint8Array([42]);
    const compressed2 = await compressData(data2);

    const bufferSize =
      4 + 8 + compressedUnhandled.length + 8 + compressed2.length;
    const buffer = new ArrayBuffer(bufferSize);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    uint8View.set([75, 72, 73, 6], 0);

    // Chunk 0: Type 99 (Unhandled / Skipped)
    dv.setUint32(4, compressedUnhandled.length, true);
    dv.setUint32(8, 99, true);
    uint8View.set(compressedUnhandled, 12);

    // Chunk 1: Type 2 (Handled, will fail during ingest)
    const offset1 = 12 + compressedUnhandled.length;
    dv.setUint32(offset1, compressed2.length, true);
    dv.setUint32(offset1 + 4, 2, true);
    uint8View.set(compressed2, offset1 + 8);

    await expectAsync(parser.parse(buffer)).toBeRejectedWithError(
      KHIChunkDecodeError,
      /Failed to decode chunk \(typeId: 2, index: 1, offset: \d+\) in version 6\./,
    );
  });

  it('should throw KHIInvalidFileError for empty buffer', async () => {
    const parser = new KHIFileParser(registry);
    const emptyBuffer = new ArrayBuffer(0);

    await expectAsync(parser.parse(emptyBuffer)).toBeRejectedWithError(
      KHIInvalidFileError,
      /Empty KHI file buffer was given/,
    );
  });

  it('should throw Error for truncated buffer smaller than header', async () => {
    const parser = new KHIFileParser(registry);
    const truncatedBuffer = new ArrayBuffer(2);

    await expectAsync(parser.parse(truncatedBuffer)).toBeRejectedWithError(
      Error,
      /Buffer too small to contain header/,
    );
  });

  it('should stream multiple chunks of the same type to the same assembler instance', async () => {
    const parser = new KHIFileParser(registry);

    const chunkA = new TextEncoder().encode('first');
    const compressedA = await compressData(chunkA);

    const chunkB = new TextEncoder().encode('second');
    const compressedB = await compressData(chunkB);

    const bufferSize = 4 + 8 + compressedA.length + 8 + compressedB.length;
    const buffer = new ArrayBuffer(bufferSize);
    const dv = new DataView(buffer);
    const uint8View = new Uint8Array(buffer);

    uint8View.set([75, 72, 73, 6], 0);

    // Chunk 0: Type 1
    dv.setUint32(4, compressedA.length, true);
    dv.setUint32(8, 1, true);
    uint8View.set(compressedA, 12);

    // Chunk 1: Type 1 again
    const offset1 = 12 + compressedA.length;
    dv.setUint32(offset1, compressedB.length, true);
    dv.setUint32(offset1 + 4, 1, true);
    uint8View.set(compressedB, offset1 + 8);

    await parser.parse(buffer);

    expect(mockAssembler1.ingest).toHaveBeenCalledTimes(2);
    expect(mockAssembler1.ingest).toHaveBeenCalledWith('first');
    expect(mockAssembler1.ingest).toHaveBeenCalledWith('second');
    // finalize should still be called only once
    expect(mockAssembler1.finalize).toHaveBeenCalledTimes(1);
  });
});
