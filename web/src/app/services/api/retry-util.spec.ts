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

import { Code, ConnectError } from '@connectrpc/connect';
import {
  calculateBackoffDelayMs,
  DEFAULT_CHUNK_MAX_RETRIES,
  DEFAULT_MAX_RETRIES,
  DEFAULT_RETRY_BASE_DELAY_MS,
  DEFAULT_RETRY_MAX_DELAY_MS,
  delayWithSignal,
  isRetryableError,
} from 'src/app/services/api/retry-util';
import { CancellationError } from 'src/app/store/domain/filter/types';

describe('retry-util', () => {
  describe('constants', () => {
    it('defines expected default retry settings', () => {
      expect(DEFAULT_MAX_RETRIES).toBe(3);
      expect(DEFAULT_CHUNK_MAX_RETRIES).toBe(10);
      expect(DEFAULT_RETRY_BASE_DELAY_MS).toBe(500);
      expect(DEFAULT_RETRY_MAX_DELAY_MS).toBe(3000);
    });
  });

  describe('isRetryableError', () => {
    it('returns false for null or undefined', () => {
      expect(isRetryableError(null)).toBeFalse();
      expect(isRetryableError(undefined)).toBeFalse();
    });

    it('returns false for CancellationError and DOMException AbortError', () => {
      expect(isRetryableError(new CancellationError('Aborted'))).toBeFalse();
      expect(
        isRetryableError(
          new DOMException('The user aborted a request', 'AbortError'),
        ),
      ).toBeFalse();
    });

    it('returns false for ConnectError with Code.Canceled', () => {
      const err = new ConnectError('User canceled', Code.Canceled);
      expect(isRetryableError(err)).toBeFalse();
    });

    it('returns true for ConnectError with Code.Unavailable', () => {
      const err = new ConnectError(
        'Service temporarily unavailable',
        Code.Unavailable,
      );
      expect(isRetryableError(err)).toBeTrue();
    });

    it('returns true for ConnectError containing 502/503/504 in message', () => {
      const err502 = new ConnectError('HTTP 502 Bad Gateway', Code.Unknown);
      const err503 = new ConnectError(
        'HTTP 503 Service Unavailable',
        Code.Unknown,
      );
      const err504 = new ConnectError('HTTP 504 Gateway Timeout', Code.Unknown);
      expect(isRetryableError(err502)).toBeTrue();
      expect(isRetryableError(err503)).toBeTrue();
      expect(isRetryableError(err504)).toBeTrue();
    });

    it('returns false for non-transient ConnectError codes', () => {
      const notFound = new ConnectError('Resource not found', Code.NotFound);
      const invalid = new ConnectError('Invalid input', Code.InvalidArgument);
      const internal = new ConnectError('Internal server panic', Code.Internal);
      expect(isRetryableError(notFound)).toBeFalse();
      expect(isRetryableError(invalid)).toBeFalse();
      expect(isRetryableError(internal)).toBeFalse();
    });

    it('returns true for TypeError network failure errors', () => {
      const fetchFailed = new TypeError('Failed to fetch');
      const networkError = new TypeError(
        'NetworkError when attempting to fetch resource.',
      );
      expect(isRetryableError(fetchFailed)).toBeTrue();
      expect(isRetryableError(networkError)).toBeTrue();
    });

    it('returns false for unrelated TypeError', () => {
      const typeErr = new TypeError('Cannot read properties of undefined');
      expect(isRetryableError(typeErr)).toBeFalse();
    });

    it('returns true for generic Error mentioning 502/503/504', () => {
      expect(
        isRetryableError(new Error('upstream server returned 502 Bad Gateway')),
      ).toBeTrue();
      expect(isRetryableError(new Error('503 Service Unavailable'))).toBeTrue();
    });

    it('returns false for generic unrelated Error', () => {
      expect(isRetryableError(new Error('Syntax error'))).toBeFalse();
    });
  });

  describe('calculateBackoffDelayMs', () => {
    it('calculates exponential delay without jitter', () => {
      expect(calculateBackoffDelayMs(1, 500, 3000, false)).toBe(500);
      expect(calculateBackoffDelayMs(2, 500, 3000, false)).toBe(1000);
      expect(calculateBackoffDelayMs(3, 500, 3000, false)).toBe(2000);
      expect(calculateBackoffDelayMs(4, 500, 3000, false)).toBe(3000); // capped at max
    });

    it('includes jitter within expected range', () => {
      const delay = calculateBackoffDelayMs(1, 500, 3000, true);
      expect(delay).toBeGreaterThanOrEqual(500);
      expect(delay).toBeLessThanOrEqual(500 + 250);
    });

    it('never exceeds maxDelayMs when jitter is added', () => {
      for (let i = 0; i < 20; i++) {
        const delay = calculateBackoffDelayMs(10, 500, 3000, true);
        expect(delay).toBeLessThanOrEqual(3000);
      }
    });
  });

  describe('delayWithSignal', () => {
    it('resolves immediately if signal is already aborted', async () => {
      const controller = new AbortController();
      controller.abort();
      const start = Date.now();
      await delayWithSignal(1000, controller.signal);
      const elapsed = Date.now() - start;
      expect(elapsed).toBeLessThan(100);
    });

    it('resolves early if signal fires during delay', async () => {
      const controller = new AbortController();
      const start = Date.now();
      const delayPromise = delayWithSignal(2000, controller.signal);
      setTimeout(() => controller.abort(), 50);
      await delayPromise;
      const elapsed = Date.now() - start;
      expect(elapsed).toBeLessThan(500);
    });

    it('resolves after specified time when not aborted', async () => {
      const start = Date.now();
      await delayWithSignal(50);
      const elapsed = Date.now() - start;
      expect(elapsed).toBeGreaterThanOrEqual(40);
    });
  });
});
