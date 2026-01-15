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

/**
 * RGB color in HDR color space.
 * All elements must be in [0, 1].
 */
export type HDRColor3 = [number, number, number];

/**
 * RGBA color in HDR color space.
 * All elements must be in [0, 1].
 */
export type HDRColor4 = [number, number, number, number];

/**
 * Utility class for converting colors and time between different formats for the renderer.
 */
export class RendererConvertUtil {
  /**
   * Convert hex RGB color to RGB.
   * Note that color space on the canvas is Display-P3 in KHI.
   * Giving sRGB hex color string will result in oversaturated colors.
   * @param hex Hex RGB color string (e.g., "#FF0000").
   * @deprecated this must be replaced with a raw HDRColor4 instead of converting sRGB color to HDRColor4 in long term.
   * @returns RGB color array ([r, g, b]).
   */
  public static hexSRGBToHDRColor(hex: string): HDRColor4 {
    if (!hex.startsWith('#')) {
      throw new Error(`Hex string must start with '#': ${hex}`);
    }
    const r = parseInt(hex.substring(1, 3), 16);
    const g = parseInt(hex.substring(3, 5), 16);
    const b = parseInt(hex.substring(5, 7), 16);
    if (hex.length === 7) {
      return [r / 255, g / 255, b / 255, 1];
    }
    const a = parseInt(hex.substring(7, 9), 16);
    return [r / 255, g / 255, b / 255, a / 255];
  }

  /**
   * Convert HDR RGB color to CSS color string.
   * @param color HDR RGB color array ([r, g, b, a] or [r, g, b]). All elements must be in [0, 1].
   * @returns CSS color string (e.g., "color(in display-p3 1 0 0 / 1)").
   */
  public static hdrColorToCSSColor(color: HDRColor4 | HDRColor3): string {
    const r = color[0];
    const g = color[1];
    const b = color[2];
    const a = color.length === 4 ? color[3] : 1;
    return `color(display-p3 ${r} ${g} ${b} / ${a})`;
  }

  /**
   * Convert HDR RGB color to CSS color string with alpha.
   * @param color HDR RGB color array ([r, g, b]). All elements must be in [0, 1].
   * @param alpha Alpha value (0-1).
   * @returns CSS color string (e.g., "color(in display-p3 1 0 0 / 1)").
   */
  public static hdrColorToCSSColorWithAlpha(
    color: HDRColor3 | HDRColor4,
    alpha: number,
  ): string {
    const r = color[0];
    const g = color[1];
    const b = color[2];
    return `color(display-p3 ${r} ${g} ${b} / ${alpha})`;
  }

  /**
   * Split float unix seconds into seconds and nano seconds.
   * @deprecated TODO: The original time held in frontend must be changed to seconds and nano seconds in integers later.
   * Converting flaot to 2 long values are not solving the precision issue. Use splitBigIntTimeToSecondsAndNanoSeconds instead.
   * @returns [seconds, nanoSeconds]
   */
  public static splitTimeToSecondsAndNanoSeconds(
    unixMillis: number,
  ): [number, number] {
    const unixSeconds = unixMillis / 1000;
    const seconds = Math.floor(unixSeconds);
    const nanoSeconds = (unixSeconds - seconds) * 1e9;
    return [seconds, nanoSeconds];
  }

  /**
   * Split bigint unix nanoseconds into seconds and nano seconds.
   * @returns [seconds, nanoSeconds]
   */
  public static splitBigIntTimeToSecondsAndNanoSeconds(
    unixNanos: bigint,
  ): [number, number] {
    const unixSeconds = unixNanos / 1000_000_000n;
    const seconds = Math.floor(Number(unixSeconds));
    const nanoSeconds = unixNanos - BigInt(seconds) * 1000_000_000n;
    return [seconds, Number(nanoSeconds)];
  }
}
