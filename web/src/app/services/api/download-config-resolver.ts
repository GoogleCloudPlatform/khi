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

import { DownloadEnvironmentConfig } from 'src/environments/environment-types';
import { environment } from 'src/environments/environment';
import {
  parseByteSizeQueryParam,
  parseIntegerQueryParam,
  resolveParam,
} from 'src/app/utils/config-resolver';

/**
 * Default chunk payload size for inspection data downloads (16 MB).
 */
export const DEFAULT_DOWNLOAD_CHUNK_SIZE_BYTES = 16 * 1024 * 1024;

/**
 * Default concurrency for inspection data downloads.
 */
export const DEFAULT_DOWNLOAD_MAX_CONCURRENCY = 10;

/**
 * Options supplied to resolve the effective download configuration.
 */
export interface ResolveDownloadConfigOptions {
  /** Optional URL search query string (defaults to window.location.search when available). */
  readonly searchString?: string;
  /** Optional explicit environment download configuration override. */
  readonly environmentConfig?: DownloadEnvironmentConfig;
}

/**
 * Effective download configuration.
 */
export interface ResolvedDownloadConfig {
  /** Effective chunk payload size in bytes. */
  readonly chunkSize: number;
  /** Effective maximum number of concurrent chunk downloads. */
  readonly maxConcurrency: number;
}

/**
 * Resolves effective chunk size and concurrency for inspection data downloads.
 *
 * Precedence for chunkSize:
 * 1. URL GET parameter `downloadChunkSize` (e.g. `?downloadChunkSize=2MB` or `?downloadChunkSize=4194304`).
 * 2. `environment.download.chunkSizeBytes`.
 * 3. Default fallback (16 MB).
 *
 * Precedence for maxConcurrency:
 * 1. URL GET parameter `downloadConcurrency` (e.g. `?downloadConcurrency=4`).
 * 2. `environment.download.maxConcurrency`.
 * 3. Default fallback (10).
 */
export function resolveDownloadConfig(
  options?: ResolveDownloadConfigOptions,
): ResolvedDownloadConfig {
  const envConfig = options?.environmentConfig ?? environment.download;

  const chunkSize = resolveParam({
    paramName: 'downloadChunkSize',
    parser: parseByteSizeQueryParam,
    searchString: options?.searchString,
    candidates: [envConfig?.chunkSizeBytes],
    defaultValue: DEFAULT_DOWNLOAD_CHUNK_SIZE_BYTES,
    validator: (v) => v > 0,
  });

  const maxConcurrency = resolveParam({
    paramName: 'downloadConcurrency',
    parser: (val) => parseIntegerQueryParam(val, 1),
    searchString: options?.searchString,
    candidates: [envConfig?.maxConcurrency],
    defaultValue: DEFAULT_DOWNLOAD_MAX_CONCURRENCY,
    validator: (v) => v > 0,
  });

  return {
    chunkSize: Math.max(1, Math.floor(chunkSize)),
    maxConcurrency: Math.max(1, Math.floor(maxConcurrency)),
  };
}
