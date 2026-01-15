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

import { RendererConvertUtil } from './convertutil';

describe('RendererConvertUtil', () => {
  describe('hexSRGBToHDRColor', () => {
    it('should convert 7-digit hex color to HDRColor4', () => {
      const result = RendererConvertUtil.hexSRGBToHDRColor('#FF0000');
      expect(result).toEqual([1, 0, 0, 1]);
    });

    it('should convert 9-digit hex color to HDRColor4', () => {
      const result = RendererConvertUtil.hexSRGBToHDRColor('#00FF0080');
      expect(result[0]).toBe(0);
      expect(result[1]).toBe(1);
      expect(result[2]).toBe(0);
      expect(result[3]).toBeCloseTo(0.5, 2);
    });
  });

  describe('hdrColorToCSSColor', () => {
    it('should convert HDRColor3 to CSS color string', () => {
      const result = RendererConvertUtil.hdrColorToCSSColor([1, 0, 0]);
      expect(result).toBe('color(display-p3 1 0 0 / 1)');
    });

    it('should convert HDRColor4 to CSS color string', () => {
      const result = RendererConvertUtil.hdrColorToCSSColor([0, 1, 0, 0.5]);
      expect(result).toBe('color(display-p3 0 1 0 / 0.5)');
    });
  });

  describe('hdrColorToCSSColorWithAlpha', () => {
    it('should convert HDRColor3 to CSS color string with specified alpha', () => {
      const result = RendererConvertUtil.hdrColorToCSSColorWithAlpha(
        [0, 0, 1],
        0.8,
      );
      expect(result).toBe('color(display-p3 0 0 1 / 0.8)');
    });

    it('should convert HDRColor4 to CSS color string with specified alpha (ignoring original alpha)', () => {
      const result = RendererConvertUtil.hdrColorToCSSColorWithAlpha(
        [0, 0, 1, 0.2],
        0.8,
      );
      expect(result).toBe('color(display-p3 0 0 1 / 0.8)');
    });
  });

  describe('splitTimeToSecondsAndNanoSeconds', () => {
    it('should split milliseconds to seconds and nanoseconds', () => {
      const result =
        RendererConvertUtil.splitTimeToSecondsAndNanoSeconds(1500.5);
      expect(result[0]).toBe(1);
      expect(result[1]).toBeCloseTo(500500000, -1); // 1.5005 * 10^9 - 1 * 10^9 = 500500000
    });
  });

  describe('splitBigIntTimeToSecondsAndNanoSeconds', () => {
    it('should split nanoseconds bigint to seconds and nanoseconds', () => {
      const result =
        RendererConvertUtil.splitBigIntTimeToSecondsAndNanoSeconds(1500500000n);
      expect(result[0]).toBe(1);
      expect(result[1]).toBe(500500000);
    });

    it('should handle large values', () => {
      const result =
        RendererConvertUtil.splitBigIntTimeToSecondsAndNanoSeconds(
          1700000000123456789n,
        );
      expect(result[0]).toBe(1700000000);
      expect(result[1]).toBe(123456789);
    });
  });
});
