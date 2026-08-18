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

import { StyleStore } from 'src/app/store/domain/style-store';

/**
 * Converts an ISO timestamp string to a nanosecond bigint.
 *
 * @param timestampString The ISO date string (e.g., '2026-05-13T00:00:00Z').
 * @returns Nanosecond timestamp as bigint.
 */
export function parseTimestampString(timestampString: string): bigint {
  const ms = new Date(timestampString).getTime();
  return BigInt(ms) * 1_000_000n;
}

/**
 * Converts an ISO timestamp string directly to Unix seconds.
 *
 * @param timestampString The ISO date string.
 * @returns Unix timestamp in seconds.
 */
export function parseUnixSeconds(timestampString: string): number {
  return Math.floor(new Date(timestampString).getTime() / 1000);
}

/**
 * Converts a hexadecimal color string to an RGBA object with values scaled 0-1.
 * Supports #RGB, #RGBA, #RRGGBB, and #RRGGBBAA formats.
 *
 * @param hex The hex color string.
 * @returns The RGBA color object.
 */
export function parseHexColor(hex: string): {
  r: number;
  g: number;
  b: number;
  a: number;
} {
  const clean = hex.replace(/^#/, '');
  let r = 0,
    g = 0,
    b = 0,
    a = 1;

  if (clean.length === 3) {
    r = parseInt(clean[0] + clean[0], 16) / 255;
    g = parseInt(clean[1] + clean[1], 16) / 255;
    b = parseInt(clean[2] + clean[2], 16) / 255;
  } else if (clean.length === 4) {
    r = parseInt(clean[0] + clean[0], 16) / 255;
    g = parseInt(clean[1] + clean[1], 16) / 255;
    b = parseInt(clean[2] + clean[2], 16) / 255;
    a = parseInt(clean[3] + clean[3], 16) / 255;
  } else if (clean.length === 6) {
    r = parseInt(clean.substring(0, 2), 16) / 255;
    g = parseInt(clean.substring(2, 4), 16) / 255;
    b = parseInt(clean.substring(4, 6), 16) / 255;
  } else if (clean.length === 8) {
    r = parseInt(clean.substring(0, 2), 16) / 255;
    g = parseInt(clean.substring(2, 4), 16) / 255;
    b = parseInt(clean.substring(4, 6), 16) / 255;
    a = parseInt(clean.substring(6, 8), 16) / 255;
  } else {
    throw new Error(`Invalid hex color format: ${hex}`);
  }

  return { r, g, b, a };
}

/**
 * Initializes the icon atlas for a given StyleStore by fetching real assets.
 * Safely aborts if assets are not hosted in the current test runner environment.
 *
 * @param styleStore The style store to populate.
 */
export async function initializeMockIconAtlas(
  styleStore: StyleStore,
): Promise<void> {
  try {
    const [imgRes, bmfontRes, codepointsRes] = await Promise.all([
      fetch('assets/zzz-material-icons-msdf.png'),
      fetch('assets/zzz-material-icons-msdf.json'),
      fetch('assets/zzz-icon-codepoints.json'),
    ]);

    if (!imgRes.ok || !bmfontRes.ok || !codepointsRes.ok) {
      console.warn(
        'Icon assets not available in current context. Skipping icon atlas initialization.',
      );
      return;
    }

    const imgBuffer = await imgRes.arrayBuffer();
    const bmfontBuffer = await bmfontRes.arrayBuffer();
    const codepointsJson = (await codepointsRes.json()) as Record<
      string,
      string
    >;

    const nameToCodepoints = new Map<string, string>(
      Object.entries(codepointsJson),
    );

    await styleStore.setIconAtlas({
      msdfIconImage: [imgBuffer],
      bmfontJson: bmfontBuffer,
      nameToCodepoints,
    });
  } catch (e) {
    console.warn(
      'Exception occurred while initializing mock icon atlas. Skipping.',
      e,
    );
  }
}
