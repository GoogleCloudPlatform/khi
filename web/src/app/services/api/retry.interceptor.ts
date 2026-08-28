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

import { Interceptor } from '@connectrpc/connect';
import {
  calculateBackoffDelayMs,
  DEFAULT_MAX_RETRIES,
  DEFAULT_RETRY_BASE_DELAY_MS,
  DEFAULT_RETRY_MAX_DELAY_MS,
  delayWithSignal,
  isRetryableError,
} from 'src/app/services/api/retry-util';
import { CancellationError } from 'src/app/store/domain/filter/types';

/**
 * Set of unary RPC method names that are idempotent and safe to retry automatically upon transient errors.
 * Chunk-level operations (GetInspectionDataChunk, UploadFileChunk, UploadInspectionChunk) are handled
 * with higher retry limits by their dedicated uploader/downloader managers and excluded here to prevent double retries.
 */
export const DEFAULT_RETRYABLE_METHODS: ReadonlySet<string> = new Set([
  'GetInspectionTypes',
  'GetInspections',
  'GetInspectionMetadata',
  'GetInspectionFeatures',
  'ReadStructYAMLs',
  'GetArchitectureGraph',
  'HeartbeatWorkbench',
  'PullIndexProgress',
  'PullServerStat',
  'PullPopup',
  'PullInspections',
  'OpenWorkbenchSync',
  'FilterTimelineSync',
  'ValidateTimelineQuery',
  'ValidateLogQuery',
]);

/**
 * Configuration options for the Connect-RPC retry interceptor.
 */
export interface RetryInterceptorOptions {
  /** Maximum number of retry attempts before giving up. Defaults to 3. */
  readonly maxRetries?: number;

  /** Base delay in milliseconds before the first retry attempt. Defaults to 500. */
  readonly baseDelayMs?: number;

  /** Maximum backoff delay in milliseconds. Defaults to 3000. */
  readonly maxDelayMs?: number;

  /** Set of RPC method names allowed to be retried. Defaults to DEFAULT_RETRYABLE_METHODS. */
  readonly retryableMethods?: ReadonlySet<string>;
}

/**
 * Creates a Connect-RPC interceptor that automatically retries idempotent unary RPCs on transient failures.
 *
 * Catches transient errors (such as HTTP 502 Bad Gateway, 503 Service Unavailable, 504 Gateway Timeout,
 * and network connection drops) and executes exponential backoff retries.
 *
 * @param options Optional configuration for retry attempts, delays, and allowed method list.
 * @returns A Connect-RPC Interceptor.
 */
export function createRetryInterceptor(
  options?: RetryInterceptorOptions,
): Interceptor {
  const maxRetries = options?.maxRetries ?? DEFAULT_MAX_RETRIES;
  const baseDelayMs = options?.baseDelayMs ?? DEFAULT_RETRY_BASE_DELAY_MS;
  const maxDelayMs = options?.maxDelayMs ?? DEFAULT_RETRY_MAX_DELAY_MS;
  const retryableMethods =
    options?.retryableMethods ?? DEFAULT_RETRYABLE_METHODS;

  return (next) => async (req) => {
    // Only intercept and retry unary calls; streaming calls manage their own lifecycle
    if (req.stream) {
      return next(req);
    }

    const isMethodAllowed = retryableMethods.has(req.method.name);
    if (!isMethodAllowed) {
      return next(req);
    }

    let retryCount = 0;
    while (true) {
      if (req.signal?.aborted) {
        return next(req);
      }

      try {
        return await next(req);
      } catch (err) {
        if (req.signal?.aborted) {
          throw new CancellationError('The operation was aborted.');
        }

        retryCount++;
        if (retryCount > maxRetries || !isRetryableError(err)) {
          throw err;
        }

        const delayMs = calculateBackoffDelayMs(
          retryCount,
          baseDelayMs,
          maxDelayMs,
        );
        await delayWithSignal(delayMs, req.signal);

        if (req.signal?.aborted) {
          throw new CancellationError('The operation was aborted.');
        }
      }
    }
  };
}
