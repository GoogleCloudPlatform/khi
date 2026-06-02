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
 * Represents inspection header metadata unpacked from the proto payload.
 */
export interface InspectionHeader {
  readonly inspectionType: string;
  readonly inspectionName: string;
  readonly inspectTimeUnixSeconds: number;
  readonly startTimeUnixSeconds: number;
  readonly endTimeUnixSeconds: number;
  readonly suggestedFilename: string;
  readonly fileSize: number;
}

/**
 * Represents a user query metadata item.
 */
export interface InspectionQuery {
  readonly id: string;
  readonly name: string;
  readonly query: string;
}

/**
 * Aggregates and exposes file-level metadata extracted from inspection chunks.
 */
export interface MetadataStore {
  /**
   * The primary inspection header details.
   */
  readonly header?: InspectionHeader;

  /**
   * List of saved inspection queries.
   */
  readonly queries: readonly InspectionQuery[];
}
