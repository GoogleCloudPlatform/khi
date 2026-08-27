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
 * Formats a duration in seconds into a human-readable display string (e.g., "45s", "180s (3m)", "90s (1.5m)").
 *
 * @param seconds - Duration in seconds.
 * @returns Human-readable duration string.
 */
export function formatDurationSeconds(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = seconds / 60;
  const minuteStr =
    seconds % 60 === 0 ? `${minutes}m` : `${minutes.toFixed(1)}m`;
  return `${seconds}s (${minuteStr})`;
}

/**
 * Generates a timestamped filename formatted as `{prefix}-YYYYMMDD-HHmmss.{extension}`.
 *
 * @param prefix - Prefix string for the filename.
 * @param extension - File format extension.
 * @param date - Date instance to derive timestamp from (defaults to now).
 * @returns Timestamped filename string.
 */
export function generateTimestampedFilename(
  prefix: string,
  extension: string,
  date = new Date(),
): string {
  const pad = (n: number): string => n.toString().padStart(2, '0');
  const yyyy = date.getFullYear();
  const mm = pad(date.getMonth() + 1);
  const dd = pad(date.getDate());
  const hh = pad(date.getHours());
  const min = pad(date.getMinutes());
  const ss = pad(date.getSeconds());
  const sanitizedExtension = extension.replace(/^\./, '');
  return `${prefix}-${yyyy}${mm}${dd}-${hh}${min}${ss}.${sanitizedExtension}`;
}
