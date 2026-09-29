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

import { Code } from '@connectrpc/connect';
import { CancellationError } from 'src/app/store/domain/filter/types';

/**
 * Default base delay in milliseconds before the first retry attempt.
 */
export const DEFAULT_RETRY_BASE_DELAY_MS = 500;

/**
 * Default maximum delay in milliseconds between retry attempts.
 */
export const DEFAULT_RETRY_MAX_DELAY_MS = 3000;

/**
 * Default maximum number of retry attempts for standard RPCs.
 */
export const DEFAULT_MAX_RETRIES = 3;

/**
 * Default maximum number of retry attempts for chunked upload and download operations.
 */
export const DEFAULT_CHUNK_MAX_RETRIES = 10;

/**
 * Checks whether an error represents a transient failure that can be safely retried.
 *
 * Transient errors include:
 * - Objects with Code.Unavailable (e.g., ConnectError, HTTP 502, 503, 504 and network drops)
 * - Browser fetch network errors (TypeError with fetch message)
 *
 * Explicit cancellations (AbortError, Code.Canceled, CancellationError) are not retryable.
 *
 * @param error The caught error to evaluate.
 * @returns True if the error is retryable, false otherwise.
 */
export function isRetryableError(error: unknown): boolean {
  if (!error) {
    return false;
  }

  if (error instanceof CancellationError) {
    return false;
  }

  if (error instanceof DOMException && error.name === 'AbortError') {
    return false;
  }

  if (typeof error === 'object' && error !== null && 'code' in error) {
    if (typeof error.code === 'number') {
      if (error.code === Code.Canceled) {
        return false;
      }
      if (error.code === Code.Unavailable) {
        return true;
      }
      if (error.code !== Code.Unknown) {
        return false;
      }
    }
  }

  if (error instanceof TypeError) {
    const message = error.message.toLowerCase();
    if (
      message.includes('failed to fetch') ||
      message.includes('networkerror') ||
      message.includes('network error') ||
      message.includes('load failed')
    ) {
      return true;
    }
  }

  if (error instanceof Error) {
    const message = error.message.toLowerCase();
    if (
      message.includes('502') ||
      message.includes('503') ||
      message.includes('504') ||
      message.includes('bad gateway') ||
      message.includes('gateway timeout') ||
      message.includes('service unavailable') ||
      message.includes('[unavailable]')
    ) {
      return true;
    }
  }

  return false;
}

/**
 * Calculates the exponential backoff delay in milliseconds for a retry attempt.
 *
 * @param retryCount The current retry attempt index (1-indexed: 1, 2, 3...).
 * @param baseDelayMs Initial delay before the first retry attempt.
 * @param maxDelayMs Upper bound for the calculated delay.
 * @param addJitter Whether to append a small random jitter to avoid thundering herds.
 * @returns Delay duration in milliseconds.
 */
export function calculateBackoffDelayMs(
  retryCount: number,
  baseDelayMs: number = DEFAULT_RETRY_BASE_DELAY_MS,
  maxDelayMs: number = DEFAULT_RETRY_MAX_DELAY_MS,
  addJitter: boolean = true,
): number {
  const attemptFactor = Math.max(0, retryCount - 1);
  const exponentialDelay = Math.min(
    maxDelayMs,
    baseDelayMs * Math.pow(2, attemptFactor),
  );
  if (!addJitter) {
    return exponentialDelay;
  }
  const jitterMs = Math.random() * (baseDelayMs * 0.5);
  return Math.min(maxDelayMs, exponentialDelay + jitterMs);
}

/**
 * Delays execution for the specified milliseconds, resolving early if the AbortSignal fires.
 *
 * @param ms Delay duration in milliseconds.
 * @param signal Optional AbortSignal to cancel waiting immediately.
 */
export function delayWithSignal(
  ms: number,
  signal?: AbortSignal,
): Promise<void> {
  return new Promise<void>((resolve) => {
    if (signal?.aborted) {
      resolve();
      return;
    }

    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, ms);

    const onAbort = () => {
      clearTimeout(timer);
      resolve();
    };

    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

/**
 * Configuration options for {@link fetchWithRetry}.
 */
export interface FetchRetryOptions {
  /** Optional RequestInit configuration passed to the underlying fetch call. */
  readonly init?: RequestInit;

  /** Maximum number of retry attempts before giving up. Defaults to DEFAULT_MAX_RETRIES (3). */
  readonly maxRetries?: number;

  /** Base delay in milliseconds before the first retry attempt. Defaults to DEFAULT_RETRY_BASE_DELAY_MS (500). */
  readonly baseDelayMs?: number;

  /** Maximum backoff delay in milliseconds. Defaults to DEFAULT_RETRY_MAX_DELAY_MS (3000). */
  readonly maxDelayMs?: number;
}

/**
 * Checks whether an HTTP status code represents a transient failure that can be safely retried.
 *
 * @param status The HTTP response status code.
 * @returns True if the status code is retryable (502, 503, 504), false otherwise.
 */
export function isRetryableHttpStatus(status: number): boolean {
  return status === 502 || status === 503 || status === 504;
}

/**
 * Executes a browser fetch request and automatically retries on transient HTTP statuses (502, 503, 504)
 * or transient network errors using exponential backoff.
 *
 * @param input The resource URL or RequestInfo to fetch.
 * @param options Optional retry and RequestInit settings.
 * @returns A promise that resolves to the fetch Response.
 */
export async function fetchWithRetry(
  input: RequestInfo | URL,
  options?: FetchRetryOptions,
): Promise<Response> {
  const maxRetries = options?.maxRetries ?? DEFAULT_MAX_RETRIES;
  const baseDelayMs = options?.baseDelayMs ?? DEFAULT_RETRY_BASE_DELAY_MS;
  const maxDelayMs = options?.maxDelayMs ?? DEFAULT_RETRY_MAX_DELAY_MS;
  const signal =
    options?.init?.signal ??
    (input instanceof Request ? input.signal : undefined);

  let retryCount = 0;
  while (true) {
    try {
      const response = options?.init
        ? await fetch(input, options.init)
        : await fetch(input);

      if (signal?.aborted) {
        throw new CancellationError('The operation was aborted.');
      }

      if (
        !response.ok &&
        isRetryableHttpStatus(response.status) &&
        retryCount < maxRetries
      ) {
        retryCount++;
        const delayMs = calculateBackoffDelayMs(
          retryCount,
          baseDelayMs,
          maxDelayMs,
        );
        await delayWithSignal(delayMs, signal);
        if (signal?.aborted) {
          throw new CancellationError('The operation was aborted.');
        }
        continue;
      }

      return response;
    } catch (err) {
      if (signal?.aborted) {
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
      await delayWithSignal(delayMs, signal);

      if (signal?.aborted) {
        throw new CancellationError('The operation was aborted.');
      }
    }
  }
}
