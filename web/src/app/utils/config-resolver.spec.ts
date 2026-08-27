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

import {
  getQueryParam,
  parseBooleanQueryParam,
  parseByteSizeQueryParam,
  parseIntegerQueryParam,
  resolveConfigValue,
  resolveParam,
} from 'src/app/utils/config-resolver';

describe('config-resolver', () => {
  describe('resolveConfigValue', () => {
    it('returns the first non-null and non-undefined candidate', () => {
      expect(
        resolveConfigValue([undefined, null, 'first', 'second'], 'default'),
      ).toBe('first');
      expect(resolveConfigValue([undefined, 0, 10], 100)).toBe(0);
      expect(resolveConfigValue([false, true], true)).toBe(false);
    });

    it('returns defaultValue when all candidates are null or undefined', () => {
      expect(resolveConfigValue([undefined, null, undefined], 'default')).toBe(
        'default',
      );
      expect(resolveConfigValue([], 42)).toBe(42);
    });
  });

  describe('getQueryParam', () => {
    it('retrieves and parses an existing query parameter', () => {
      const result = getQueryParam(
        'count',
        (v) => parseInt(v, 10),
        '?count=42&other=foo',
      );
      expect(result).toBe(42);
    });

    it('returns undefined if parameter is not in search string', () => {
      const result = getQueryParam(
        'missing',
        (v) => parseInt(v, 10),
        '?count=42',
      );
      expect(result).toBeUndefined();
    });

    it('returns undefined if parser returns undefined', () => {
      const result = getQueryParam('count', () => undefined, '?count=invalid');
      expect(result).toBeUndefined();
    });
  });

  describe('resolveParam', () => {
    it('resolves query parameter when present and valid', () => {
      const result = resolveParam({
        paramName: 'limit',
        parser: (v) => parseIntegerQueryParam(v, 1),
        searchString: '?limit=20',
        candidates: [10],
        defaultValue: 5,
      });
      expect(result).toBe(20);
    });

    it('resolves candidate when query parameter is missing or invalid', () => {
      const result = resolveParam({
        paramName: 'limit',
        parser: (v) => parseIntegerQueryParam(v, 1),
        searchString: '?limit=invalid',
        candidates: [10],
        defaultValue: 5,
      });
      expect(result).toBe(10);
    });

    it('falls back to defaultValue when query param and candidates are undefined', () => {
      const result = resolveParam({
        paramName: 'limit',
        parser: (v) => parseIntegerQueryParam(v, 1),
        searchString: '',
        candidates: [undefined, null],
        defaultValue: 5,
      });
      expect(result).toBe(5);
    });
  });

  describe('parseBooleanQueryParam', () => {
    it('parses truthy values correctly', () => {
      expect(parseBooleanQueryParam('true')).toBeTrue();
      expect(parseBooleanQueryParam('TRUE')).toBeTrue();
      expect(parseBooleanQueryParam('1')).toBeTrue();
      expect(parseBooleanQueryParam('yes')).toBeTrue();
      expect(parseBooleanQueryParam('YES')).toBeTrue();
      expect(parseBooleanQueryParam('  true  ')).toBeTrue();
    });

    it('parses falsy values correctly', () => {
      expect(parseBooleanQueryParam('false')).toBeFalse();
      expect(parseBooleanQueryParam('FALSE')).toBeFalse();
      expect(parseBooleanQueryParam('0')).toBeFalse();
      expect(parseBooleanQueryParam('no')).toBeFalse();
      expect(parseBooleanQueryParam('NO')).toBeFalse();
      expect(parseBooleanQueryParam('  false  ')).toBeFalse();
    });

    it('returns undefined for invalid boolean values', () => {
      expect(parseBooleanQueryParam(null)).toBeUndefined();
      expect(parseBooleanQueryParam(undefined)).toBeUndefined();
      expect(parseBooleanQueryParam('')).toBeUndefined();
      expect(parseBooleanQueryParam('foo')).toBeUndefined();
      expect(parseBooleanQueryParam('2')).toBeUndefined();
    });
  });

  describe('parseIntegerQueryParam', () => {
    it('parses positive integers correctly', () => {
      expect(parseIntegerQueryParam('10')).toBe(10);
      expect(parseIntegerQueryParam('  5  ')).toBe(5);
    });

    it('enforces min constraint', () => {
      expect(parseIntegerQueryParam('0', 1)).toBeUndefined();
      expect(parseIntegerQueryParam('-5', 1)).toBeUndefined();
      expect(parseIntegerQueryParam('3', 5)).toBeUndefined();
      expect(parseIntegerQueryParam('5', 5)).toBe(5);
    });

    it('returns undefined for non-numeric inputs', () => {
      expect(parseIntegerQueryParam(null)).toBeUndefined();
      expect(parseIntegerQueryParam(undefined)).toBeUndefined();
      expect(parseIntegerQueryParam('')).toBeUndefined();
      expect(parseIntegerQueryParam('abc')).toBeUndefined();
    });
  });

  describe('parseByteSizeQueryParam', () => {
    it('parses plain byte counts', () => {
      expect(parseByteSizeQueryParam('1024')).toBe(1024);
      expect(parseByteSizeQueryParam('1048576')).toBe(1048576);
    });

    it('parses unit suffixes case-insensitively', () => {
      expect(parseByteSizeQueryParam('1KB')).toBe(1024);
      expect(parseByteSizeQueryParam('2kb')).toBe(2048);
      expect(parseByteSizeQueryParam('4kib')).toBe(4096);
      expect(parseByteSizeQueryParam('1MB')).toBe(1024 * 1024);
      expect(parseByteSizeQueryParam('2mb')).toBe(2 * 1024 * 1024);
      expect(parseByteSizeQueryParam('16MiB')).toBe(16 * 1024 * 1024);
      expect(parseByteSizeQueryParam('1GB')).toBe(1024 * 1024 * 1024);
    });

    it('parses decimal values with units', () => {
      expect(parseByteSizeQueryParam('1.5MB')).toBe(
        Math.floor(1.5 * 1024 * 1024),
      );
      expect(parseByteSizeQueryParam('0.5KB')).toBe(512);
    });

    it('handles spaces between number and unit', () => {
      expect(parseByteSizeQueryParam('2 MB')).toBe(2 * 1024 * 1024);
      expect(parseByteSizeQueryParam(' 512  KB ')).toBe(512 * 1024);
    });

    it('returns undefined for invalid inputs', () => {
      expect(parseByteSizeQueryParam(null)).toBeUndefined();
      expect(parseByteSizeQueryParam(undefined)).toBeUndefined();
      expect(parseByteSizeQueryParam('')).toBeUndefined();
      expect(parseByteSizeQueryParam('abc')).toBeUndefined();
      expect(parseByteSizeQueryParam('10XB')).toBeUndefined();
      expect(parseByteSizeQueryParam('-5MB')).toBeUndefined();
      expect(parseByteSizeQueryParam('0MB')).toBeUndefined();
    });
  });
});
