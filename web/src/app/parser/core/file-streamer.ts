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

import { InspectionData, TimeRange } from 'src/app/store/inspection-data';
import { NullReferenceResolver } from 'src/app/common/loader/reference-resolver';
import {
  KHIInvalidFileError,
  KHIVersionMismatchError,
  KHIChunkDecodeError,
  KHIDataAssemblyError,
} from 'src/app/parser/errors/parser-errors';
import {
  IDataAssembler,
  ParserBlueprint,
} from 'src/app/parser/core/interfaces';
import { BinaryReader } from 'src/app/parser/core/binary-reader';

/**
 * Orchestrator class responsible for streaming and parsing KHI inspection files.
 */
export class KHIFileStreamer {
  private static readonly MAGIC = 'KHI';

  /**
   * Creates a new instance of the KHIFileStreamer.
   */
  constructor(
    private readonly versionRegistry: Record<number, ParserBlueprint>,
  ) {}

  /**
   * Parses the raw KHI binary file into the final UI View Model.
   */
  async parse(buffer: ArrayBuffer): Promise<InspectionData> {
    const reader = new BinaryReader(buffer);

    // 1. Header Validation
    const { magic, version } = reader.readHeader();
    if (magic !== KHIFileStreamer.MAGIC) {
      throw new KHIInvalidFileError('Invalid magic bytes. Not a KHI file.');
    }

    // 2. Blueprint Resolution
    const blueprint = this.versionRegistry[version];
    if (!blueprint) {
      throw new KHIVersionMismatchError(
        `Unsupported KHI file version: ${version}`,
      );
    }

    const activeAssemblers = new Map<number, IDataAssembler<unknown>>();
    for (const [typeId, definition] of blueprint.entries()) {
      activeAssemblers.set(typeId, definition.createAssembler());
    }

    let chunkIndex = 0;

    // 3. Chunk Streaming & Ingestion Phase
    while (reader.hasMore()) {
      const offset = reader.currentOffset;
      const { size, typeId, data } = await reader.readNextChunk();

      const assembler = activeAssemblers.get(typeId);
      if (!assembler) {
        throw new Error(`Unknown chunk type ${typeId} at offset ${offset}.`);
      }

      try {
        const definition = blueprint.get(typeId)!;
        // Decode raw bytes to Protobuf, then ingest into the assembler
        const proto = definition.decode(data);
        assembler.ingest(proto);
      } catch (error) {
        throw new KHIChunkDecodeError({
          version,
          typeId,
          chunkIndex,
          offset,
          cause: error,
        });
      }

      chunkIndex++;
    }

    // 4. Priority-Based Assembly Phase
    // The assembly logic and domain stores instantiation (InspectionDataBuilder)
    // will be implemented in a subsequent stage.
    return null as unknown as InspectionData;
  }
}
