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
  formatDurationSeconds,
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
});
