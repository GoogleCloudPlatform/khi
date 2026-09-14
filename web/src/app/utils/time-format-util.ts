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

/**
 * Formats a unix timestamp in seconds to an ISO 8601 string taking timezone shift into account.
 *
 * @param timestampSeconds - Unix timestamp in seconds.
 * @param timezoneShiftHours - Timezone offset from UTC in hours (e.g. 9 for UTC+09:00).
 * @returns Formatted ISO 8601 string, or '-' if timestamp is non-positive or non-finite.
 */
export function formatIsoTimestampSeconds(
  timestampSeconds: number,
  timezoneShiftHours: number,
): string {
  if (timestampSeconds <= 0 || !Number.isFinite(timestampSeconds)) {
    return '-';
  }
  const sign = timezoneShiftHours >= 0 ? '+' : '-';
  const absHours = Math.abs(timezoneShiftHours);
  const shiftHour = Math.floor(absHours);
  const shiftMinute = Math.round((absHours % 1) * 60);
  const shiftHourStr = shiftHour.toString().padStart(2, '0');
  const shiftMinuteStr = shiftMinute.toString().padStart(2, '0');
  const offsetStr = `${sign}${shiftHourStr}:${shiftMinuteStr}`;

  const shiftedDate = new Date(
    (timestampSeconds + timezoneShiftHours * 3600) * 1000,
  );
  const pad = (n: number): string => n.toString().padStart(2, '0');
  const yyyy = shiftedDate.getUTCFullYear();
  const mm = pad(shiftedDate.getUTCMonth() + 1);
  const dd = pad(shiftedDate.getUTCDate());
  const hh = pad(shiftedDate.getUTCHours());
  const min = pad(shiftedDate.getUTCMinutes());
  const ss = pad(shiftedDate.getUTCSeconds());

  return `${yyyy}-${mm}-${dd}T${hh}:${min}:${ss}${offsetStr}`;
}
