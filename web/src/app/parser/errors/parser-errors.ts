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

/**
 * Thrown when the file does not have the expected KHI magic bytes.
 */
export class KHIInvalidFileError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'KHIInvalidFileError';
  }
}

/**
 * Thrown when the parsed file version is not supported by any registered blueprint.
 */
export class KHIVersionMismatchError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'KHIVersionMismatchError';
  }
}

/**
 * Context provided when a chunk fails to decode.
 */
export interface ChunkErrorContext {
  readonly version: number;
  readonly typeId: number;
  readonly chunkIndex: number;
  readonly offset: number;
  readonly cause: unknown;
}

/**
 * Thrown when a specific chunk fails to decode from Protobuf binary.
 */
export class KHIChunkDecodeError extends Error {
  constructor(public readonly context: ChunkErrorContext) {
    super(
      `Failed to decode chunk (typeId: ${context.typeId}, index: ${context.chunkIndex}, offset: ${context.offset}) in version ${context.version}.`,
      { cause: context.cause },
    );
    this.name = 'KHIChunkDecodeError';
  }
}

/**
 * Context provided when the data assembly phase fails.
 */
export interface AssemblyErrorContext {
  readonly version: number;
  readonly typeId: number;
  readonly cause: unknown;
}

/**
 * Thrown when an assembler fails to mutate the final model.
 */
export class KHIDataAssemblyError extends Error {
  constructor(public readonly context: AssemblyErrorContext) {
    super(
      `Failed to assemble data for chunk type ${context.typeId} in version ${context.version}.`,
      { cause: context.cause },
    );
    this.name = 'KHIDataAssemblyError';
  }
}
