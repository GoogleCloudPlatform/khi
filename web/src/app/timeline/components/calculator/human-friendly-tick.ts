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

/**
 * Function type for converting a timestamp (in nanoseconds) to a human-readable string label.
 */
export type NsTimeToLabelConverter = (ns: bigint) => string;

const formatHHMMSSSSS = (ns: bigint) => {
  const d = new Date(BigIntTimeUtil.NsToNumberMs(ns));
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  const s = d.getUTCSeconds().toString().padStart(2, '0');
  const S = d.getUTCMilliseconds().toString().padStart(3, '0');
  return `${h}:${m}:${s}.${S}`;
};

const formatHHMMSS = (ns: bigint) => {
  const d = new Date(BigIntTimeUtil.NsToNumberMs(ns));
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  const s = d.getUTCSeconds().toString().padStart(2, '0');
  return `${h}:${m}:${s}`;
};

const formatHHMM = (ns: bigint) => {
  const d = new Date(BigIntTimeUtil.NsToNumberMs(ns));
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  return `${h}:${m}`;
};

interface HumanFriendlyStep {
  ns: bigint;
  labelConverter: NsTimeToLabelConverter;
  minimumTimeLabelSpaceInPx: number;
}

/**
 * Definition of a step in the ruler.
 * Describes how to divide the timeline and format labels for a specific zoom level.
 */
export interface RulerStepDefinition {
  /** The time interval (in ns) for the smallest tick. */
  low: bigint;
  /** The multiplier to get the next level of ticks (middle importance). */
  middleMultiplier: bigint;
  /** The multiplier to get the highest level of ticks (high importance). */
  highMultiplier: bigint;
  /** The function to convert time (ns) to a label string. */
  labelConverter: NsTimeToLabelConverter;
  /** The minimum space (px) required for the time label. */
  minimumTimeLabelSpaceInPx: number;
}

const MS = 1_000_000n;
const SEC = 1000n * MS;
const MIN = 60n * SEC;
const HOUR = 60n * MIN;

const DAY = 24n * HOUR;

const HUMAN_FRIENDLY_STEPS: HumanFriendlyStep[] = [
  // ms
  {
    ns: 1n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ns: 5n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ns: 10n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ns: 50n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ns: 100n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  {
    ns: 500n * MS,
    labelConverter: formatHHMMSSSSS,
    minimumTimeLabelSpaceInPx: 500,
  },
  // sec
  {
    ns: 1n * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  {
    ns: 5n * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  {
    ns: 10n * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  {
    ns: 30n * SEC,
    labelConverter: formatHHMMSS,
    minimumTimeLabelSpaceInPx: 300,
  },
  // min
  { ns: 1n * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ns: 5n * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ns: 10n * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ns: 30n * MIN, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  // hour
  { ns: 1n * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ns: 3n * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  { ns: 6n * HOUR, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
  {
    ns: 12n * HOUR,
    labelConverter: formatHHMM,
    minimumTimeLabelSpaceInPx: 200,
  },
  { ns: 1n * DAY, labelConverter: formatHHMM, minimumTimeLabelSpaceInPx: 200 },
];

/**
 * A list of ruler steps derived from human-friendly intervals.
 * Used to determine the appropriate tick interval for the current zoom level.
 */
export const RULER_STEPS_NS: RulerStepDefinition[] = [];

for (let i = 0; i < HUMAN_FRIENDLY_STEPS.length - 2; i++) {
  const low = HUMAN_FRIENDLY_STEPS[i];
  const middle = HUMAN_FRIENDLY_STEPS[i + 1];
  const high = HUMAN_FRIENDLY_STEPS[i + 2];
  RULER_STEPS_NS.push({
    low: low.ns,
    middleMultiplier: middle.ns / low.ns,
    highMultiplier: high.ns / low.ns,
    labelConverter: high.labelConverter,
    minimumTimeLabelSpaceInPx: high.minimumTimeLabelSpaceInPx,
  });
}

/**
 * Calculates the best time span (bucket size) for a histogram based on the query time range and maximum histogram size.
 * It tries to find a human-friendly step that fits within the max histogram size.
 *
 * @param maxHistogramSize The maximum number of buckets allowed in the histogram.
 * @param minQueryTimeNs The start time of the query range (ns).
 * @param maxQueryTimeNs The end time of the query range (ns).
 * @returns The selected time span (in ns) for each histogram bucket.
 */
export function getMinTimeSpanForHistogram(
  maxHistogramSize: number,
  minQueryTimeNs: bigint,
  maxQueryTimeNs: bigint,
): bigint {
  const timeSpanNs = maxQueryTimeNs - minQueryTimeNs;
  const maxBuckets = BigInt(maxHistogramSize);
  return (
    RULER_STEPS_NS.find((step) => timeSpanNs <= step.low * maxBuckets)?.low ??
    timeSpanNs
  );
}

/**
 * Returns the time interval between ticks (ns) for the current zoom level.
 *
 * @param pixelsPerNs The current zoom level in pixels per nanosecond.
 * @returns The tick time interval in nanoseconds.
 */
export function getTickTimeNs(
  pixelsPerNs: number,
  minGridWidthPx: number,
): bigint {
  const step = getRulerStep(pixelsPerNs, minGridWidthPx);
  return step.low;
}

/**
 * Determines the appropriate ruler step (tick interval) based on the zoom level.
 *
 * @param pixelsPerNs The current zoom level in pixels per nanosecond.
 * @returns The selected RulerStepDefinition.
 */
export function getRulerStep(
  pixelsPerNs: number,
  minGridWidthPx: number,
): RulerStepDefinition {
  const minTimeSpanNs = minGridWidthPx / pixelsPerNs;
  return (
    RULER_STEPS_NS.find((step) => Number(step.low) >= minTimeSpanNs) ??
    RULER_STEPS_NS[RULER_STEPS_NS.length - 1]
  );
}
