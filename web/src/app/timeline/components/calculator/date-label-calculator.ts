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
 * Represents a date label on the timeline ruler at a specific day boundary.
 *
 * This label indicates the transition between two days. For example, exactly at the
 * midnight boundary between March 11 and March 12, `labelLeft` will be 'YYYY/MM/11'
 * and `labelRight` will be 'YYYY/MM/12'.
 */
export interface DateLabel {
  /** The horizontal offset from the left edge of the timeline in pixels representing the boundary time. */
  offsetX: number;
  /** The date string (YYYY/MM/DD) representing the day immediately before the boundary. */
  labelLeft: string;
  /** The date string (YYYY/MM/DD) representing the day starting exactly at the boundary. */
  labelRight: string;
}

/**
 * Calculates the positions and formatted text for date labels (e.g., "YYYY/MM/DD") based on the current
 * visible time range on the timeline ruler. This ensures that day boundaries crossing the view
 * are correctly labeled.
 *
 * The calculation mathematically aligns to the exact local time midnight boundary by factoring in
 * the given timezone shift, ensuring that the visual representation maps exactly to the viewer's local day.
 *
 * @param leftEdgeTime The timestamp (in UTC milliseconds) corresponding to the visible left edge of the timeline.
 * @param visibleDuration The total duration (in milliseconds) visible on the screen.
 * @param pixelsPerMs The current zoom level, expressed as pixels per millisecond.
 * @param timezoneShiftHours The timezone offset in hours to apply. Positive values move ahead of UTC, negative values move behind.
 * @returns An array of `DateLabel` objects representing the day boundaries within the visible range.
 */
export function calculateDateLabels(
  leftEdgeTime: number,
  visibleDuration: number,
  pixelsPerMs: number,
  timezoneShiftHours: number,
): DateLabel[] {
  const labels: DateLabel[] = [];
  const DAY = 60 * 60 * 24 * 1000;
  const timezoneShiftInMs = timezoneShiftHours * 60 * 60 * 1000;

  const localLeftEdgeTime = leftEdgeTime + timezoneShiftInMs;
  const localPrevMidnight = Math.floor(localLeftEdgeTime / DAY) * DAY - DAY;
  let prevDayTime = localPrevMidnight - timezoneShiftInMs;

  while (true) {
    const currentDayTime = prevDayTime + DAY;
    if (currentDayTime > leftEdgeTime + visibleDuration) {
      break;
    }
    labels.push({
      offsetX: (currentDayTime - leftEdgeTime) * pixelsPerMs,
      labelLeft: toDateLabel(currentDayTime - 1, timezoneShiftHours),
      labelRight: toDateLabel(currentDayTime, timezoneShiftHours),
    });
    prevDayTime = currentDayTime;
  }
  return labels;
}

/**
 * Formats a given timestamp into a date string (YYYY/MM/DD) according to the configured timezone shift.
 *
 * @param time The timestamp in milliseconds to format.
 * @param timezoneShiftHours The timezone offset in hours applied to the formatting.
 * @returns The formatted date string.
 */
function toDateLabel(time: number, timezoneShiftHours: number): string {
  const date = new Date(time + timezoneShiftHours * 60 * 60 * 1000);
  const year = date.getUTCFullYear();
  const month = ('' + (date.getUTCMonth() + 1)).padStart(2, '0');
  const day = ('' + date.getUTCDate()).padStart(2, '0');
  return `${year}/${month}/${day}`;
}
