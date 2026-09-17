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
 * Formats a duration in milliseconds into a compact display string such as "820ms", "4.3s", or "2m05s".
 *
 * @param durationMs - Duration in milliseconds.
 * @returns Compact duration string.
 */
export function formatDurationMs(durationMs: number): string {
  if (durationMs < 1000) {
    return `${Math.round(durationMs)}ms`;
  }
  if (durationMs < 60000) {
    return `${(durationMs / 1000).toFixed(1)}s`;
  }
  const totalSeconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}m${seconds.toString().padStart(2, '0')}s`;
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
  const totalOffsetMinutes = Math.round(Math.abs(timezoneShiftHours) * 60);
  const shiftHour = Math.floor(totalOffsetMinutes / 60);
  const shiftMinute = totalOffsetMinutes % 60;
  const shiftHourStr = shiftHour.toString().padStart(2, '0');
  const shiftMinuteStr = shiftMinute.toString().padStart(2, '0');
  const offsetStr = `${sign}${shiftHourStr}:${shiftMinuteStr}`;

  const signedOffsetMinutes =
    timezoneShiftHours >= 0 ? totalOffsetMinutes : -totalOffsetMinutes;
  const shiftedDate = new Date(
    (timestampSeconds + signedOffsetMinutes * 60) * 1000,
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

/**
 * Formats a timestamp in nanoseconds to an ISO 8601 string with millisecond precision taking timezone shift into account.
 * Format: `YYYY-MM-DDTHH:mm:ss.SSS+HH:mm` (or `-HH:mm`).
 *
 * @param timestampNs - Unix timestamp in nanoseconds as bigint.
 * @param timezoneShiftHours - Timezone offset from UTC in hours (e.g. 9 for UTC+09:00).
 * @returns Formatted ISO 8601 string, or '-' if timestampNs <= 0n.
 */
export function formatIsoTimestampNs(
  timestampNs: bigint,
  timezoneShiftHours: number,
): string {
  if (timestampNs <= 0n) {
    return '-';
  }
  const sign = timezoneShiftHours >= 0 ? '+' : '-';
  const totalOffsetMinutes = Math.round(Math.abs(timezoneShiftHours) * 60);
  const shiftHour = Math.floor(totalOffsetMinutes / 60);
  const shiftMinute = totalOffsetMinutes % 60;
  const shiftHourStr = shiftHour.toString().padStart(2, '0');
  const shiftMinuteStr = shiftMinute.toString().padStart(2, '0');
  const offsetStr = `${sign}${shiftHourStr}:${shiftMinuteStr}`;

  const signedOffsetMinutes =
    timezoneShiftHours >= 0 ? totalOffsetMinutes : -totalOffsetMinutes;

  const msBigInt = timestampNs / 1000000n;
  const subMsNs = Number(timestampNs % 1000000n);
  const roundedMs = subMsNs >= 500000 ? Number(msBigInt) + 1 : Number(msBigInt);

  const shiftedDate = new Date(roundedMs + signedOffsetMinutes * 60 * 1000);
  const pad = (n: number): string => n.toString().padStart(2, '0');
  const pad3 = (n: number): string => n.toString().padStart(3, '0');
  const yyyy = shiftedDate.getUTCFullYear();
  const mm = pad(shiftedDate.getUTCMonth() + 1);
  const dd = pad(shiftedDate.getUTCDate());
  const hh = pad(shiftedDate.getUTCHours());
  const min = pad(shiftedDate.getUTCMinutes());
  const ss = pad(shiftedDate.getUTCSeconds());
  const sss = pad3(shiftedDate.getUTCMilliseconds());

  return `${yyyy}-${mm}-${dd}T${hh}:${min}:${ss}.${sss}${offsetStr}`;
}

/**
 * Parses an ISO 8601 string into a timestamp in nanoseconds (bigint), applying timezoneShiftHours if no timezone is specified.
 *
 * @param input - Input ISO 8601 string (e.g. "2026-09-17T14:00:00Z", "2026-09-17 14:00:00", "2026-09-17T14:00:00+09:00").
 * @param timezoneShiftHours - Timezone offset from UTC in hours to apply when input has no timezone suffix.
 * @returns Timestamp in nanoseconds as bigint, or null if input is empty or invalid.
 */
export function parseIsoTimestampNs(
  input: string,
  timezoneShiftHours: number,
): bigint | null {
  const trimmed = input.trim();
  if (!trimmed) {
    return null;
  }

  // Check if trimmed already has a timezone indicator (Z, +HH:mm, -HH:mm, +HHmm, -HHmm, +HH, -HH) after a time component
  const hasTimezone =
    /(?:T|\s)\d{2}(?::\d{2})*(?:\.\d+)?\s*(?:Z|[+-]\d{2}(?::?\d{2})?)$/i.test(
      trimmed,
    );

  let dateStringToParse = trimmed;
  if (!hasTimezone) {
    const sign = timezoneShiftHours >= 0 ? '+' : '-';
    const totalOffsetMinutes = Math.round(Math.abs(timezoneShiftHours) * 60);
    const shiftHour = Math.floor(totalOffsetMinutes / 60);
    const shiftMinute = totalOffsetMinutes % 60;
    const offsetStr = `${sign}${shiftHour.toString().padStart(2, '0')}:${shiftMinute.toString().padStart(2, '0')}`;
    // Replace space between date and time with T if present so Date.parse accepts it reliably
    const normalized = trimmed.replace(' ', 'T');
    const timePart = normalized.includes('T') ? '' : 'T00:00:00';
    dateStringToParse = `${normalized}${timePart}${offsetStr}`;
  } else {
    dateStringToParse = trimmed.replace(' ', 'T');
  }

  const parsedMs = Date.parse(dateStringToParse);
  if (isNaN(parsedMs)) {
    return null;
  }

  return BigInt(Math.round(parsedMs)) * 1000000n;
}

/**
 * Formats a time range [startTimeNs, endTimeNs] into a compact chip label in the shifted timezone.
 *
 * @param startTimeNs - Start timestamp in nanoseconds.
 * @param endTimeNs - End timestamp in nanoseconds.
 * @param timezoneShiftHours - Timezone offset from UTC in hours.
 * @returns Compact chip label (e.g. "YYYY-MM-DD HH:mm:ss ~ HH:mm:ss" if same day, or "YYYY-MM-DD HH:mm:ss ~ YYYY-MM-DD HH:mm:ss" if different days).
 */
export function formatTimeRangeChipLabel(
  startTimeNs: bigint,
  endTimeNs: bigint,
  timezoneShiftHours: number,
): string {
  if (startTimeNs <= 0n || endTimeNs <= 0n) {
    return '-';
  }
  const totalOffsetMinutes = Math.round(Math.abs(timezoneShiftHours) * 60);
  const signedOffsetMinutes =
    timezoneShiftHours >= 0 ? totalOffsetMinutes : -totalOffsetMinutes;

  const startMs = Number(startTimeNs / 1000000n);
  const endMs = Number(endTimeNs / 1000000n);

  const startDate = new Date(startMs + signedOffsetMinutes * 60 * 1000);
  const endDate = new Date(endMs + signedOffsetMinutes * 60 * 1000);

  const pad = (n: number): string => n.toString().padStart(2, '0');

  const startYMD = `${startDate.getUTCFullYear()}-${pad(startDate.getUTCMonth() + 1)}-${pad(startDate.getUTCDate())}`;
  const startHMS = `${pad(startDate.getUTCHours())}:${pad(startDate.getUTCMinutes())}:${pad(startDate.getUTCSeconds())}`;

  const endYMD = `${endDate.getUTCFullYear()}-${pad(endDate.getUTCMonth() + 1)}-${pad(endDate.getUTCDate())}`;
  const endHMS = `${pad(endDate.getUTCHours())}:${pad(endDate.getUTCMinutes())}:${pad(endDate.getUTCSeconds())}`;

  if (startYMD === endYMD) {
    return `${startYMD} ${startHMS} ~ ${endHMS}`;
  }
  return `${startYMD} ${startHMS} ~ ${endYMD} ${endHMS}`;
}
