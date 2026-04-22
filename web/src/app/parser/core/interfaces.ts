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

import { InspectionData } from 'src/app/store/inspection-data';

/**
 * Stateful assembler that collects decoded Protobufs and mutates the final model.
 */
export interface IDataAssembler<TProto = unknown> {
  /**
   * Ingests a decoded Protobuf chunk. Called multiple times if chunks are split.
   */
  ingest(proto: TProto): void;

  /**
   * Integrates the ingested data into the final InspectionData model.
   */
  assembleInto(model: InspectionData): void;
}

/**
 * Defines how a specific chunk type is handled for a specific version.
 */
export interface ChunkDefinition<TProto = unknown> {
  readonly typeId: number;
  /**
   * Stateless function to decode raw bytes into a Protobuf object.
   */
  readonly decode: (bytes: Uint8Array) => TProto;
  /**
   * Factory method for the stateful assembler.
   */
  readonly createAssembler: () => IDataAssembler<TProto>;
  /**
   * Execution priority for dependency resolution (Lower number = executed first).
   */
  readonly priority: number;
}

/**
 * A version-specific registry of chunk definitions.
 */
export type ParserBlueprint = Map<number, ChunkDefinition>;
