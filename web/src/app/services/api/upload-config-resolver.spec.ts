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

import { resolveUploadConfig } from 'src/app/services/api/upload-config-resolver';
import { parseByteSizeQueryParam as parseByteSizeString } from 'src/app/utils/config-resolver';
import {
  DEFAULT_CHUNK_SIZE_BYTES,
  DEFAULT_MAX_CONCURRENCY,
} from 'src/app/services/api/chunked-uploader';

describe('upload-config-resolver', () => {
  describe('parseByteSizeString', () => {
    it('returns undefined for empty, null, or undefined input', () => {
      expect(parseByteSizeString(null)).toBeUndefined();
      expect(parseByteSizeString(undefined)).toBeUndefined();
      expect(parseByteSizeString('')).toBeUndefined();
      expect(parseByteSizeString('   ')).toBeUndefined();
    });

    it('parses plain byte numbers', () => {
      expect(parseByteSizeString('1024')).toBe(1024);
      expect(parseByteSizeString('16777216')).toBe(16777216);
      expect(parseByteSizeString('  500  ')).toBe(500);
    });

    it('parses byte sizes with unit suffixes case-insensitively', () => {
      expect(parseByteSizeString('512K')).toBe(512 * 1024);
      expect(parseByteSizeString('512kb')).toBe(512 * 1024);
      expect(parseByteSizeString('512KiB')).toBe(512 * 1024);
      expect(parseByteSizeString('2M')).toBe(2 * 1024 * 1024);
      expect(parseByteSizeString('2mb')).toBe(2 * 1024 * 1024);
      expect(parseByteSizeString('2MiB')).toBe(2 * 1024 * 1024);
      expect(parseByteSizeString('1G')).toBe(1024 * 1024 * 1024);
      expect(parseByteSizeString('1gb')).toBe(1024 * 1024 * 1024);
      expect(parseByteSizeString('1GiB')).toBe(1024 * 1024 * 1024);
      expect(parseByteSizeString('2048B')).toBe(2048);
    });

    it('handles decimal values with units', () => {
      expect(parseByteSizeString('1.5MB')).toBe(Math.round(1.5 * 1024 * 1024));
      expect(parseByteSizeString('0.5KB')).toBe(Math.round(0.5 * 1024));
    });

    it('handles whitespace between numbers and units', () => {
      expect(parseByteSizeString('4 MB')).toBe(4 * 1024 * 1024);
      expect(parseByteSizeString(' 128  kb ')).toBe(128 * 1024);
    });

    it('returns undefined for non-positive or invalid strings', () => {
      expect(parseByteSizeString('0')).toBeUndefined();
      expect(parseByteSizeString('0MB')).toBeUndefined();
      expect(parseByteSizeString('-10MB')).toBeUndefined();
      expect(parseByteSizeString('abc')).toBeUndefined();
      expect(parseByteSizeString('invalidMB')).toBeUndefined();
    });
  });

  describe('resolveUploadConfig', () => {
    it('returns defaults when no options or environment configurations are provided', () => {
      const config = resolveUploadConfig({
        searchString: '',
        environmentConfig: {},
      });

      expect(config.chunkSize).toBe(DEFAULT_CHUNK_SIZE_BYTES);
      expect(config.maxConcurrency).toBe(DEFAULT_MAX_CONCURRENCY);
    });

    it('uses backend suggestedChunkSizeBytes when provided and no overrides exist', () => {
      const config = resolveUploadConfig({
        suggestedChunkSizeBytes: 25 * 1024 * 1024,
        searchString: '',
        environmentConfig: {},
      });

      expect(config.chunkSize).toBe(25 * 1024 * 1024);
      expect(config.maxConcurrency).toBe(DEFAULT_MAX_CONCURRENCY);
    });

    it('prefers environment config over backend suggestedChunkSizeBytes and default concurrency', () => {
      const config = resolveUploadConfig({
        suggestedChunkSizeBytes: 25 * 1024 * 1024,
        searchString: '',
        environmentConfig: {
          chunkSizeBytes: 8 * 1024 * 1024,
          maxConcurrency: 6,
        },
      });

      expect(config.chunkSize).toBe(8 * 1024 * 1024);
      expect(config.maxConcurrency).toBe(6);
    });

    it('prefers callerMaxConcurrency over environment config', () => {
      const config = resolveUploadConfig({
        callerMaxConcurrency: 3,
        searchString: '',
        environmentConfig: {
          maxConcurrency: 8,
        },
      });

      expect(config.maxConcurrency).toBe(3);
    });

    it('prefers GET query parameters over environment config, server suggestions, and caller options', () => {
      const config = resolveUploadConfig({
        suggestedChunkSizeBytes: 25 * 1024 * 1024,
        callerMaxConcurrency: 2,
        searchString: '?uploadChunkSize=1MB&uploadConcurrency=10',
        environmentConfig: {
          chunkSizeBytes: 8 * 1024 * 1024,
          maxConcurrency: 4,
        },
      });

      expect(config.chunkSize).toBe(1024 * 1024);
      expect(config.maxConcurrency).toBe(10);
    });

    it('falls back to environment config if GET parameters are invalid', () => {
      const config = resolveUploadConfig({
        suggestedChunkSizeBytes: 25 * 1024 * 1024,
        searchString: '?uploadChunkSize=invalid&uploadConcurrency=notanumber',
        environmentConfig: {
          chunkSizeBytes: 4 * 1024 * 1024,
          maxConcurrency: 5,
        },
      });

      expect(config.chunkSize).toBe(4 * 1024 * 1024);
      expect(config.maxConcurrency).toBe(5);
    });

    it('clamps chunkSize and maxConcurrency to at least 1', () => {
      const config = resolveUploadConfig({
        searchString: '?uploadConcurrency=0',
        suggestedChunkSizeBytes: 0,
        callerMaxConcurrency: 0,
        environmentConfig: {
          chunkSizeBytes: 0,
          maxConcurrency: 0,
        },
      });

      expect(config.chunkSize).toBeGreaterThanOrEqual(1);
      expect(config.maxConcurrency).toBeGreaterThanOrEqual(1);
    });
  });
});
