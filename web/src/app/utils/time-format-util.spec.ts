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
  formatIsoTimestampNs,
  formatIsoTimestampSeconds,
  formatTimeRangeChipLines,
  formatTimeRangeTooltip,
  generateTimestampedFilename,
  parseIsoTimestampNs,
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
    });

    it('should format durations of one hour or more but less than a day in hours and minutes', () => {
      expect(formatDurationMs(3600000)).toBe('1h00m');
      expect(formatDurationMs(11 * 3600000 + 5 * 60000)).toBe('11h05m');
    });

    it('should format durations of one day or more in days and hours', () => {
      expect(formatDurationMs(86400000)).toBe('1d00h');
      expect(formatDurationMs(2 * 86400000 + 3 * 3600000)).toBe('2d03h');
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

  describe('formatIsoTimestampNs', () => {
    // 1700000000123000000n is 2023-11-14T22:13:20.123Z
    const timestampNs = 1700000000123000000n;

    it('should format UTC timestamp with millisecond precision when timezone shift is 0', () => {
      expect(formatIsoTimestampNs(timestampNs, 0)).toBe(
        '2023-11-14T22:13:20.123+00:00',
      );
    });

    it('should format timestamp with positive timezone shift (+9 for JST)', () => {
      expect(formatIsoTimestampNs(timestampNs, 9)).toBe(
        '2023-11-15T07:13:20.123+09:00',
      );
    });

    it('should format timestamp with negative timezone shift (-5 for EST)', () => {
      expect(formatIsoTimestampNs(timestampNs, -5)).toBe(
        '2023-11-14T17:13:20.123-05:00',
      );
    });

    it('should format timestamp with fractional timezone shift (+5.5 for IST)', () => {
      expect(formatIsoTimestampNs(timestampNs, 5.5)).toBe(
        '2023-11-15T03:43:20.123+05:30',
      );
    });

    it('should return "-" when timestampNs <= 0n', () => {
      expect(formatIsoTimestampNs(0n, 9)).toBe('-');
      expect(formatIsoTimestampNs(-100n, 9)).toBe('-');
    });
  });

  describe('parseIsoTimestampNs', () => {
    it('should return null for empty or whitespace-only input', () => {
      expect(parseIsoTimestampNs('', 0)).toBeNull();
      expect(parseIsoTimestampNs('   ', 9)).toBeNull();
    });

    it('should parse ISO 8601 string with explicit UTC "Z"', () => {
      const parsed = parseIsoTimestampNs('2023-11-14T22:13:20.123Z', 9);
      expect(parsed).toBe(1700000000123000000n);
    });

    it('should parse ISO 8601 string with explicit timezone offset', () => {
      const parsed = parseIsoTimestampNs('2023-11-15T07:13:20.123+09:00', 0);
      expect(parsed).toBe(1700000000123000000n);
    });

    it('should apply timezoneShiftHours when no timezone suffix is present', () => {
      // With +9 shift, "2023-11-15T07:13:20.123" represents UTC 2023-11-14T22:13:20.123
      const parsed = parseIsoTimestampNs('2023-11-15T07:13:20.123', 9);
      expect(parsed).toBe(1700000000123000000n);
    });

    it('should support space separator between date and time', () => {
      const parsed = parseIsoTimestampNs('2023-11-15 07:13:20.123', 9);
      expect(parsed).toBe(1700000000123000000n);
    });

    it('should correctly apply timezone shift for date-only string', () => {
      // 2023-11-15 with +9 shift -> 2023-11-15T00:00:00+09:00 -> UTC 2023-11-14T15:00:00Z = 1699974000s
      const parsed = parseIsoTimestampNs('2023-11-15', 9);
      expect(parsed).toBe(1699974000000000000n);
    });

    it('should return null for invalid date strings', () => {
      expect(parseIsoTimestampNs('invalid-date', 0)).toBeNull();
      expect(parseIsoTimestampNs('2023-99-99T99:99:99', 0)).toBeNull();
    });
  });

  describe('formatTimeRangeChipLines', () => {
    // 2023-11-15 07:00:00 JST (+9) -> UTC 2023-11-14 22:00:00
    const startSameDay = 1699999200000000000n;
    // 2023-11-15 09:30:00 JST (+9) -> UTC 2023-11-15 00:30:00
    const endSameDay = 1700008200000000000n;
    // 2023-11-16 08:00:00 JST (+9) -> UTC 2023-11-15 23:00:00
    const endDifferentDay = 1700089200000000000n;

    it('should format two lines when start and end fall on the same day', () => {
      const lines = formatTimeRangeChipLines(startSameDay, endSameDay, 9);
      expect(lines).toEqual({
        startLine: '2023-11-15 07:00:00',
        endLine: '~ 09:30:00',
      });
    });

    it('should include the date in endLine when start and end fall on different days', () => {
      const lines = formatTimeRangeChipLines(startSameDay, endDifferentDay, 9);
      expect(lines).toEqual({
        startLine: '2023-11-15 07:00:00',
        endLine: '~ 2023-11-16 08:00:00',
      });
    });

    it('should respect timezone shift', () => {
      // In UTC (shift 0), start is 2023-11-14 22:00:00 and end is 2023-11-15 00:30:00 (different days)
      const linesUtc = formatTimeRangeChipLines(startSameDay, endSameDay, 0);
      expect(linesUtc).toEqual({
        startLine: '2023-11-14 22:00:00',
        endLine: '~ 2023-11-15 00:30:00',
      });

      // In UTC-5 (shift -5), start is 2023-11-14 17:00:00 and end is 2023-11-14 19:30:00 (same day)
      const linesEst = formatTimeRangeChipLines(startSameDay, endSameDay, -5);
      expect(linesEst).toEqual({
        startLine: '2023-11-14 17:00:00',
        endLine: '~ 19:30:00',
      });
    });

    it('should return "-" fallback when either timestamp is non-positive', () => {
      expect(formatTimeRangeChipLines(0n, endSameDay, 9)).toEqual({
        startLine: '-',
        endLine: '',
      });
      expect(formatTimeRangeChipLines(startSameDay, 0n, 9)).toEqual({
        startLine: '-',
        endLine: '',
      });
      expect(formatTimeRangeChipLines(-1n, endSameDay, 9)).toEqual({
        startLine: '-',
        endLine: '',
      });
      expect(formatTimeRangeChipLines(startSameDay, -1n, 9)).toEqual({
        startLine: '-',
        endLine: '',
      });
    });
  });

  describe('formatTimeRangeTooltip', () => {
    const start = 1699999200000000000n;
    const end = 1700008200000000000n;

    it('should format detailed tooltip label with ISO timestamps and duration', () => {
      expect(formatTimeRangeTooltip(start, end, 9)).toBe(
        '2023-11-15T07:00:00.000+09:00 ~ 2023-11-15T09:30:00.000+09:00 (2h30m)',
      );
    });

    it('should derive the duration from the same rounded milliseconds as the rendered timestamps', () => {
      // The start rounds down and the end rounds up, so a truncating duration would be 1ms short.
      const startWithSubMs = 1700000000000400000n;
      const endWithSubMs = 1700000000820600000n;
      expect(formatTimeRangeTooltip(startWithSubMs, endWithSubMs, 0)).toBe(
        '2023-11-14T22:13:20.000+00:00 ~ 2023-11-14T22:13:20.821+00:00 (821ms)',
      );
    });

    it('should return "-" when either timestamp is non-positive', () => {
      expect(formatTimeRangeTooltip(0n, end, 9)).toBe('-');
      expect(formatTimeRangeTooltip(start, 0n, 9)).toBe('-');
      expect(formatTimeRangeTooltip(-100n, end, 9)).toBe('-');
      expect(formatTimeRangeTooltip(start, -100n, 9)).toBe('-');
    });
  });
});
