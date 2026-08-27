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

import { UploadEnvironmentConfig } from 'src/environments/environment-types';
import { environment } from 'src/environments/environment';
import {
  DEFAULT_CHUNK_SIZE_BYTES,
  DEFAULT_MAX_CONCURRENCY,
} from 'src/app/services/api/chunked-uploader';
import {
  parseByteSizeQueryParam,
  parseIntegerQueryParam,
  resolveParam,
} from 'src/app/utils/config-resolver';

/**
 * Options supplied to resolve the effective upload configuration.
 */
export interface ResolveUploadConfigOptions {
  /** Optional chunk size suggested by the backend server in bytes. */
  readonly suggestedChunkSizeBytes?: number;
  /** Optional max concurrency requested by the programmatic caller. */
  readonly callerMaxConcurrency?: number;
  /** Optional URL search query string (defaults to window.location.search when available). */
  readonly searchString?: string;
  /** Optional environment upload config (defaults to environment.upload). */
  readonly environmentConfig?: UploadEnvironmentConfig;
}

/**
 * Resolved chunk upload parameters.
 */
export interface ResolvedUploadConfig {
  /** The effective chunk size in bytes (clamped to >= 1). */
  readonly chunkSize: number;
  /** The effective maximum concurrency limit (clamped to >= 1). */
  readonly maxConcurrency: number;
}

/**
 * Resolves chunk upload configuration with priority:
 *
 * For chunkSize:
 * 1. URL GET parameter `uploadChunkSize` (bytes or human-readable format like `2MB`, `512KB`).
 * 2. `environment.upload.chunkSizeBytes`.
 * 3. Server `suggestedChunkSizeBytes` (from StartImportInspection / StartFileUpload).
 * 4. Fallback default (16 MB).
 *
 * For maxConcurrency:
 * 1. URL GET parameter `uploadConcurrency` (integer).
 * 2. Programmatic caller options (`callerMaxConcurrency`).
 * 3. `environment.upload.maxConcurrency`.
 * 4. Fallback default (4).
 */
export function resolveUploadConfig(
  options?: ResolveUploadConfigOptions,
): ResolvedUploadConfig {
  const envConfig = options?.environmentConfig ?? environment.upload;

  const chunkSize = resolveParam({
    paramName: 'uploadChunkSize',
    parser: parseByteSizeQueryParam,
    searchString: options?.searchString,
    candidates: [envConfig?.chunkSizeBytes, options?.suggestedChunkSizeBytes],
    defaultValue: DEFAULT_CHUNK_SIZE_BYTES,
    validator: (v) => v > 0,
  });

  const maxConcurrency = resolveParam({
    paramName: 'uploadConcurrency',
    parser: (val) => parseIntegerQueryParam(val, 1),
    searchString: options?.searchString,
    candidates: [options?.callerMaxConcurrency, envConfig?.maxConcurrency],
    defaultValue: DEFAULT_MAX_CONCURRENCY,
    validator: (v) => v > 0,
  });

  return {
    chunkSize: Math.max(1, Math.floor(chunkSize)),
    maxConcurrency: Math.max(1, Math.floor(maxConcurrency)),
  };
}
