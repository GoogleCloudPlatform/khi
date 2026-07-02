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

import { InspectionData } from 'src/app/store/domain/inspection-data';
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
import { InspectionDataBuilder } from 'src/app/parser/core/builder';
import { ProgressReporter } from 'src/app/services/progress/progress-interface';

/**
 * Orchestrator class responsible for streaming and parsing KHI inspection files.
 */
export class KHIFileParser {
  private static readonly MAGIC = 'KHI';
  private static readonly PROGRESS_INITIALIZED = 0;
  private static readonly PROGRESS_BLUEPRINT_RESOLVED = 5;
  private static readonly PROGRESS_INGESTION_START =
    KHIFileParser.PROGRESS_BLUEPRINT_RESOLVED;
  private static readonly PROGRESS_INGESTION_END = 80;
  private static readonly PROGRESS_ASSEMBLY_START =
    KHIFileParser.PROGRESS_INGESTION_END;
  private static readonly PROGRESS_ASSEMBLY_END = 95;
  private static readonly PROGRESS_BUILD_START =
    KHIFileParser.PROGRESS_ASSEMBLY_END;
  private static readonly PROGRESS_COMPLETED = 100;

  /**
   * Creates a new instance of the KHIFileParser.
   */
  constructor(
    private readonly versionRegistry: Record<number, ParserBlueprint>,
  ) {}

  /**
   * Parses the raw KHI binary file into the final UI View Model.
   * @param buffer The raw binary file data.
   * @param progressReporter Optional reporter to receive parsing progress updates.
   */
  async parse(
    buffer: ArrayBuffer,
    progressReporter?: ProgressReporter,
  ): Promise<InspectionData> {
    if (buffer.byteLength === 0) {
      throw new KHIInvalidFileError('Empty KHI file buffer was given');
    }
    progressReporter?.reportProgress(KHIFileParser.PROGRESS_INITIALIZED);
    progressReporter?.reportMessage('Initializing KHI file parsing...');
    const reader = new BinaryReader(buffer);

    // 1. Header Validation
    const { magic, version } = reader.readHeader();
    if (magic !== KHIFileParser.MAGIC) {
      throw new KHIInvalidFileError('Invalid magic bytes. Not a KHI file.');
    }

    // 2. Blueprint Resolution
    progressReporter?.reportProgress(KHIFileParser.PROGRESS_BLUEPRINT_RESOLVED);
    progressReporter?.reportMessage('Resolving blueprint...');
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
    const executedTypeIds = new Set<number>();

    // 3. Chunk Streaming & Ingestion Phase
    while (reader.hasMore()) {
      const offset = reader.currentOffset;
      const { typeId, data } = await reader.readNextChunk();

      const currentRatio = reader.currentOffset / buffer.byteLength;
      const currentPercent = Math.min(
        KHIFileParser.PROGRESS_INGESTION_END - 1,
        Math.floor(
          KHIFileParser.PROGRESS_INGESTION_START +
            (KHIFileParser.PROGRESS_INGESTION_END -
              KHIFileParser.PROGRESS_INGESTION_START) *
              currentRatio,
        ),
      );
      progressReporter?.reportProgress(currentPercent);
      progressReporter?.reportMessage(
        `Parsing (${formatBytes(reader.currentOffset)} / ${formatBytes(buffer.byteLength)})...`,
      );

      const assembler = activeAssemblers.get(typeId);
      if (!assembler) {
        throw new Error(`Unknown chunk type ${typeId} at offset ${offset}.`);
      }

      executedTypeIds.add(typeId);

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
    progressReporter?.reportProgress(KHIFileParser.PROGRESS_ASSEMBLY_START);
    progressReporter?.reportMessage('Preparing data assembly...');
    const executedDefinitions = Array.from(executedTypeIds).map(
      (typeId) => blueprint.get(typeId)!,
    );
    executedDefinitions.sort((a, b) => a.priority - b.priority);

    const builder = new InspectionDataBuilder();
    let defIndex = 0;
    for (const def of executedDefinitions) {
      const assembler = activeAssemblers.get(def.typeId)!;
      const assemblyPercent = Math.floor(
        KHIFileParser.PROGRESS_ASSEMBLY_START +
          ((KHIFileParser.PROGRESS_ASSEMBLY_END -
            KHIFileParser.PROGRESS_ASSEMBLY_START) *
            defIndex) /
            executedDefinitions.length,
      );
      progressReporter?.reportProgress(assemblyPercent);
      progressReporter?.reportMessage(`Assembling ${def.label}...`);
      try {
        assembler.assembleInto(builder);
      } catch (error) {
        throw new KHIDataAssemblyError({
          version,
          typeId: def.typeId,
          cause: error,
        });
      }
      defIndex++;
    }

    progressReporter?.reportProgress(KHIFileParser.PROGRESS_BUILD_START);
    progressReporter?.reportMessage('Building inspection data...');
    const result = await builder.build();
    progressReporter?.reportProgress(KHIFileParser.PROGRESS_COMPLETED);
    progressReporter?.complete();
    return result;
  }
}

/**
 * Formats a number of bytes into a human-readable string with units (e.g., "1.50 MB").
 * @param bytes The number of bytes to format.
 * @returns The formatted string.
 */
function formatBytes(bytes: number): string {
  if (bytes <= 0) {
    return '0 B';
  }
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const formatted = (bytes / Math.pow(k, i)).toFixed(2);
  return `${formatted} ${sizes[i]}`;
}
