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
 * Configuration options for chunked uploads defined in environments.
 */
export interface UploadEnvironmentConfig {
  /** Default chunk payload size in bytes. */
  readonly chunkSizeBytes?: number;
  /** Maximum number of concurrent chunk uploads. */
  readonly maxConcurrency?: number;
}

/**
 * Configuration options for chunked downloads defined in environments.
 */
export interface DownloadEnvironmentConfig {
  /** Default chunk payload size in bytes. */
  readonly chunkSizeBytes?: number;
  /** Maximum number of concurrent chunk downloads. */
  readonly maxConcurrency?: number;
}
