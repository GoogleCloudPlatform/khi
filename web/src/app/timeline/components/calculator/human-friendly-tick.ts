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
 * Function type for converting a timestamp (in milliseconds) to a human-readable string label.
 */
export type MSTimeToLabelConverter = (ms: number) => string;

const formatHHMMSSSSS = (ms: number) => {
  const d = new Date(ms);
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  const s = d.getUTCSeconds().toString().padStart(2, '0');
  const S = d.getUTCMilliseconds().toString().padStart(3, '0');
  return `${h}:${m}:${s}.${S}`;
};

const formatHHMMSS = (ms: number) => {
  const d = new Date(ms);
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  const s = d.getUTCSeconds().toString().padStart(2, '0');
  return `${h}:${m}:${s}`;
};

const formatHHMM = (ms: number) => {
  const d = new Date(ms);
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  return `${h}:${m}`;
};

interface HumanFriendlyStep {
  ms: number;
  labelConverter: MSTimeToLabelConverter;
  minimumTimeLabelSpaceInPx: number;
}

/**
 * Definition of a step in the ruler.
 * Describes how to divide the timeline and format labels for a specific zoom level.
 */
export interface RulerStepDefinition {
  /** The time interval (in ms) for the smallest tick. */
  low: number;
  /** The multiplier to get the next level of ticks (middle importance). */
  middleMultiplier: number;
  /** The multiplier to get the highest level of ticks (high importance). */
  highMultiplier: number;
  /** The function to convert time (ms) to a label string. */
  labelConverter: MSTimeToLabelConverter;
  /** The minimum space (px) required for the time label. */
  minimumTimeLabelSpaceInPx: number;
}

const MS = 1;
const SEC = 1000 * MS;
const MIN = 60 * SEC;
const HOUR = 60 * MIN;

const DAY = 24 * HOUR;

const HUMAN_FRIENDLY_STEPS: HumanFriendlyStep[] = [
  // ms
  {
    ms: 1 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ms: 5 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ms: 10 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ms: 50 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ms: 100 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ms: 500 * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  // sec
  { ms: 1 * SEC, labelConverter: formatHHMMSS, minimumTimeLabelSpaceInPx: 300 },
  { ms: 5 * SEC, labelConverter: formatHHMMSS, minimumTimeLabelSpaceInPx: 300 },
  {
    ms: 10 * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  {
    ms: 30 * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  // min
  { ms: 1 * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 5 * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 10 * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 30 * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  // hour
  { ms: 1 * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 3 * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 6 * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 12 * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ms: 1 * DAY, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
];

/**
 * A list of ruler steps derived from human-friendly intervals.
 * Used to determine the appropriate tick interval for the current zoom level.
 */
export const RULER_STEPS_MS: RulerStepDefinition[] = [];

for (let i = 0; i < HUMAN_FRIENDLY_STEPS.length - 2; i++) {
  const low = HUMAN_FRIENDLY_STEPS[i];
  const middle = HUMAN_FRIENDLY_STEPS[i + 1];
  const high = HUMAN_FRIENDLY_STEPS[i + 2];
  RULER_STEPS_MS.push({
    low: low.ms,
    middleMultiplier: Math.round(middle.ms / low.ms),
    highMultiplier: Math.round(high.ms / low.ms),
    labelConverter: high.labelConverter,
    minimumTimeLabelSpaceInPx: high.minimumTimeLabelSpaceInPx,
  });
}

/**
 * Calculates the best time span (bucket size) for a histogram based on the query time range and maximum histogram size.
 * It tries to find a human-friendly step that fits within the max histogram size.
 *
 * @param maxHistogramSize The maximum number of buckets allowed in the histogram.
 * @param minQueryTime The start time of the query range (ms).
 * @param maxQueryTime The end time of the query range (ms).
 * @returns The selected time span (in ms) for each histogram bucket.
 */
export function getMinTimeSpanForHistogram(
  maxHistogramSize: number,
  minQueryTime: number,
  maxQueryTime: number,
): number {
  const timeSpan = maxQueryTime - minQueryTime;
  return (
    RULER_STEPS_MS.find((step) => timeSpan / step.low <= maxHistogramSize)
      ?.low ?? timeSpan
  );
}

/**
 * Returns the time interval between ticks (ms) for the current zoom level.
 *
 * @param pixelsPerMs The current zoom level in pixels per millisecond.
 * @returns The tick time interval in milliseconds.
 */
export function getTickTimeMS(
  pixelsPerMs: number,
  minGridWidthPx: number,
): number {
  const step = getRulerStep(pixelsPerMs, minGridWidthPx);
  return step.low;
}

/**
 * Determines the appropriate ruler step (tick interval) based on the zoom level.
 *
 * @param pixelsPerMs The current zoom level in pixels per millisecond.
 * @returns The selected RulerStepDefinition.
 */
export function getRulerStep(
  pixelsPerMs: number,
  minGridWidthPx: number,
): RulerStepDefinition {
  const minTimeSpanMS = minGridWidthPx / pixelsPerMs;
  return (
    RULER_STEPS_MS.find((step) => step.low >= minTimeSpanMS) ??
    RULER_STEPS_MS[RULER_STEPS_MS.length - 1]
  );
}
