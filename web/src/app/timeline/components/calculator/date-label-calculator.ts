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

import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';

function floorDiv(a: bigint, b: bigint): bigint {
  const q = a / b;
  const r = a % b;
  return r !== 0n && a < 0n !== b < 0n ? q - 1n : q;
}

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
 * @param leftEdgeTimeNs The timestamp (in UTC nanoseconds) corresponding to the visible left edge of the timeline.
 * @param visibleDurationNs The total duration (in nanoseconds) visible on the screen.
 * @param pixelsPerNs The current zoom level, expressed as pixels per nanosecond.
 * @param timezoneShiftHours The timezone offset in hours to apply. Positive values move ahead of UTC, negative values move behind.
 * @returns An array of `DateLabel` objects representing the day boundaries within the visible range.
 */
export function calculateDateLabels(
  leftEdgeTimeNs: bigint,
  visibleDurationNs: bigint,
  pixelsPerNs: number,
  timezoneShiftHours: number,
): DateLabel[] {
  const labels: DateLabel[] = [];
  const DAY_NS = 24n * 60n * 60n * 1_000_000_000n;
  const timezoneShiftInNs = BigInt(
    Math.round(timezoneShiftHours * 60 * 60 * 1_000_000_000),
  );

  const localLeftEdgeTimeNs = leftEdgeTimeNs + timezoneShiftInNs;
  const localPrevMidnightNs =
    floorDiv(localLeftEdgeTimeNs, DAY_NS) * DAY_NS - DAY_NS;
  let prevDayTimeNs = localPrevMidnightNs - timezoneShiftInNs;

  while (true) {
    const currentDayTimeNs = prevDayTimeNs + DAY_NS;
    if (currentDayTimeNs > leftEdgeTimeNs + visibleDurationNs) {
      break;
    }
    labels.push({
      offsetX: Number(currentDayTimeNs - leftEdgeTimeNs) * pixelsPerNs,
      labelLeft: toDateLabel(currentDayTimeNs - 1_000_000n, timezoneShiftInNs),
      labelRight: toDateLabel(currentDayTimeNs, timezoneShiftInNs),
    });
    prevDayTimeNs = currentDayTimeNs;
  }
  return labels;
}

/**
 * Formats a given timestamp into a date string (YYYY/MM/DD) according to the configured timezone shift.
 *
 * @param timeNs The timestamp in nanoseconds to format.
 * @param timezoneShiftInNs The timezone offset in nanoseconds applied to the formatting.
 * @returns The formatted date string.
 */
function toDateLabel(timeNs: bigint, timezoneShiftInNs: bigint): string {
  const date = new Date(
    BigIntTimeUtil.NsToNumberMs(timeNs + timezoneShiftInNs),
  );
  const year = date.getUTCFullYear();
  const month = ('' + (date.getUTCMonth() + 1)).padStart(2, '0');
  const day = ('' + date.getUTCDate()).padStart(2, '0');
  return `${year}/${month}/${day}`;
}
