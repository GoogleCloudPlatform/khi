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
  formatDurationMs,
  formatDurationSeconds,
  formatIsoTimestampSeconds,
  generateTimestampedFilename,
} from './time-format-util';

describe('time-format-util', () => {
  describe('formatDurationSeconds', () => {
    it('should format seconds less than 60 including boundary values', () => {
      expect(formatDurationSeconds(0)).toBe('0s');
      expect(formatDurationSeconds(10)).toBe('10s');
      expect(formatDurationSeconds(45)).toBe('45s');
      expect(formatDurationSeconds(59)).toBe('59s');
    });

    it('should format whole minutes without decimals', () => {
      expect(formatDurationSeconds(60)).toBe('60s (1m)');
      expect(formatDurationSeconds(180)).toBe('180s (3m)');
      expect(formatDurationSeconds(3600)).toBe('3600s (60m)');
    });

    it('should format fractional minutes with one decimal place', () => {
      expect(formatDurationSeconds(90)).toBe('90s (1.5m)');
      expect(formatDurationSeconds(100)).toBe('100s (1.7m)');
    });
  });

  describe('formatDurationMs', () => {
    it('should format sub second durations in milliseconds', () => {
      expect(formatDurationMs(0)).toBe('0ms');
      expect(formatDurationMs(12.4)).toBe('12ms');
      expect(formatDurationMs(999)).toBe('999ms');
    });

    it('should format sub minute durations in seconds with one decimal place', () => {
      expect(formatDurationMs(1000)).toBe('1.0s');
      expect(formatDurationMs(4321)).toBe('4.3s');
      expect(formatDurationMs(59999)).toBe('60.0s');
    });

    it('should format longer durations in minutes with zero padded seconds', () => {
      expect(formatDurationMs(60000)).toBe('1m00s');
      expect(formatDurationMs(125000)).toBe('2m05s');
      expect(formatDurationMs(3600000)).toBe('60m00s');
    });
  });

  describe('generateTimestampedFilename', () => {
    it('should format filename correctly with given date and extension', () => {
      const fixedDate = new Date(2026, 7, 27, 15, 30, 45); // August 27, 2026 15:30:45
      const svgFilename = generateTimestampedFilename(
        'khi-graph',
        'svg',
        fixedDate,
      );
      expect(svgFilename).toBe('khi-graph-20260827-153045.svg');

      const pngFilename = generateTimestampedFilename(
        'khi-graph',
        'png',
        fixedDate,
      );
      expect(pngFilename).toBe('khi-graph-20260827-153045.png');
    });

    it('should pad single-digit months, days, and times', () => {
      const singleDigitDate = new Date(2026, 0, 5, 3, 4, 5); // Jan 5, 2026 03:04:05
      const filename = generateTimestampedFilename(
        'test',
        'json',
        singleDigitDate,
      );
      expect(filename).toBe('test-20260105-030405.json');
    });

    it('should strip leading dot from extension', () => {
      const fixedDate = new Date(2026, 7, 27, 15, 30, 45);
      const filename = generateTimestampedFilename(
        'khi-graph',
        '.svg',
        fixedDate,
      );
      expect(filename).toBe('khi-graph-20260827-153045.svg');
    });
  });

  describe('formatIsoTimestampSeconds', () => {
    // 1700000000 is 2023-11-14T22:13:20Z
    const timestampSeconds = 1700000000;

    it('should format UTC timestamp when timezone shift is 0', () => {
      expect(formatIsoTimestampSeconds(timestampSeconds, 0)).toBe(
        '2023-11-14T22:13:20+00:00',
      );
    });

    it('should format timestamp with positive integer timezone shift (+9 for JST)', () => {
      expect(formatIsoTimestampSeconds(timestampSeconds, 9)).toBe(
        '2023-11-15T07:13:20+09:00',
      );
    });

    it('should format timestamp with negative integer timezone shift (-5 for EST)', () => {
      expect(formatIsoTimestampSeconds(timestampSeconds, -5)).toBe(
        '2023-11-14T17:13:20-05:00',
      );
    });

    it('should format timestamp with positive fractional timezone shift (+5.5 for IST)', () => {
      expect(formatIsoTimestampSeconds(timestampSeconds, 5.5)).toBe(
        '2023-11-15T03:43:20+05:30',
      );
    });

    it('should handle floating point near-hour offsets without rounding minutes to 60', () => {
      expect(
        formatIsoTimestampSeconds(timestampSeconds, 5.999999999999999),
      ).toBe('2023-11-15T04:13:20+06:00');
    });

    it('should format timestamp with negative fractional timezone shift (-3.5 for NST)', () => {
      expect(formatIsoTimestampSeconds(timestampSeconds, -3.5)).toBe(
        '2023-11-14T18:43:20-03:30',
      );
    });

    it('should zero-pad single-digit months, days, hours, minutes, and seconds', () => {
      // 1704423845 is 2024-01-05T03:04:05Z
      expect(formatIsoTimestampSeconds(1704423845, 0)).toBe(
        '2024-01-05T03:04:05+00:00',
      );
    });

    it('should return "-" for zero, negative, or non-finite timestamp values', () => {
      expect(formatIsoTimestampSeconds(0, 9)).toBe('-');
      expect(formatIsoTimestampSeconds(-1, 9)).toBe('-');
      expect(formatIsoTimestampSeconds(NaN, 9)).toBe('-');
      expect(formatIsoTimestampSeconds(Infinity, 9)).toBe('-');
    });
  });
});
