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

import { HistogramCache } from './misc/histogram-cache';
import {
  getRulerStep,
  NsTimeToLabelConverter,
} from './calculator/human-friendly-tick';

function floorDiv(a: bigint, b: bigint): bigint {
  const q = a / b;
  const r = a % b;
  return r !== 0n && a < 0n !== b < 0n ? q - 1n : q;
}

/**
 * The view model for the timeline ruler(header part of the timeline component).
 */
export interface TimelineRulerViewModel {
  /**
   * The time duration of the smallest scale step in nanoseconds.
   */
  tickTimeNs: bigint;

  /**
   * The time duration of a histogram bucket in nanoseconds.
   * This must be divisible by tickTimeNs to align the histogram and scales.
   */
  histogramBucketTimeNs: bigint;

  /**
   * The start time of the histogram in nanoseconds.
   */
  histogramBeginTimeNs: bigint;

  /**
   * The histogram buckets.
   */
  histogramBuckets: HistogramBucketViewModel[];

  /**
   * The ruler ticks containing information for each scale marks on the ruler.
   */
  ticks: RulerTickViewModel[];

  /**
   * The time labels placed on the ruler.
   */
  timeLabels: RulerTimeLabelViewModel[];
}

/**
 * The time label placed on the ruler.
 */
export interface RulerTimeLabelViewModel {
  offsetLeftInPx: number;
  label: string;
}

/**
 * containing information for each scale marks on the ruler.
 */
export interface RulerTickViewModel {
  leftEdgeTimeImportance: TickImportance;
}

/**
 * A histogram bucket containing the ratio of log counts for each severity.
 */
export interface HistogramBucketViewModel {
  /**
   * [0-1] ratio of the log count in the bucket for each severity.
   */
  all: { [severityId: number]: number };

  /**
   * [0-1] ratio of the highlighted log count in the bucket for each severity.
   */
  highlighted: { [severityId: number]: number };
}

/**
 * The importance of a tick. This is used to determine the style of the tick rendered on the header.
 */
export enum TickImportance {
  Low,
  Middle,
  High,
}

/**
 * RulerViewModelBuilder calculates the view model for the timeline ruler.
 * It determines the appropriate time step (tick interval) based on the zoom level,
 * generates ticks, time labels, and histogram buckets for the ruler.
 */
export class RulerViewModelBuilder {
  /**
   * @param extraOffsetWidthInPx Extra width in pixels to render outside the viewport for buffering.
   * @param maxHistogramSize Maximum number of buckets for the histogram.
   */
  constructor(
    readonly extraOffsetWidthInPx: number = 300,
    readonly minTickWidthInPx: number = 10,
  ) {}

  /**
   * Generates the complete view model for the timeline ruler, including ticks, histogram bars, and time labels.
   *
   * @param leftEdgeTimeNs The time at the left edge of the viewport (ns).
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @param viewportWidth The width of the viewport (px).
   * @param timezoneShiftHours The timezone offset in hours to adjust time labels.
   * @param allLogsHistogramCache Cache for all logs histogram data.
   * @param filteredLogsHistogramCache Cache for filtered logs histogram data.
   * @returns The generated TimelineRulerViewModel.
   */
  generateRulerViewModel(
    leftEdgeTimeNs: bigint,
    pixelsPerNs: number,
    viewportWidth: number,
    timezoneShiftHours: number,
    allLogsHistogramCache: HistogramCache,
    filteredLogsHistogramCache: HistogramCache,
  ): TimelineRulerViewModel {
    const step = getRulerStep(pixelsPerNs, this.minTickWidthInPx);

    const tickTimeNs = step.low;
    const startTickIndex = floorDiv(leftEdgeTimeNs, tickTimeNs);

    const numberOfTicks =
      Math.ceil(
        (viewportWidth + this.extraOffsetWidthInPx * 2) /
          (Number(tickTimeNs) * pixelsPerNs),
      ) + 1; // +1 for buffer
    const ticks: RulerTickViewModel[] = new Array(numberOfTicks);
    for (let i = 0; i < numberOfTicks; i++) {
      const currentTickIndex = startTickIndex + BigInt(i);
      let importance = TickImportance.Low;
      if (currentTickIndex % step.highMultiplier === 0n) {
        importance = TickImportance.High;
      } else if (currentTickIndex % step.middleMultiplier === 0n) {
        importance = TickImportance.Middle;
      }
      ticks[i] = {
        leftEdgeTimeImportance: importance,
      };
    }

    const histogramData = allLogsHistogramCache.getHistogramData(
      startTickIndex * tickTimeNs,
      (startTickIndex + BigInt(numberOfTicks)) * tickTimeNs,
      tickTimeNs,
    );
    const filteredHistogramData = filteredLogsHistogramCache.getHistogramData(
      startTickIndex * tickTimeNs,
      (startTickIndex + BigInt(numberOfTicks)) * tickTimeNs,
      tickTimeNs,
      histogramData.totalLogCount,
    );
    const windows: HistogramBucketViewModel[] = new Array(
      histogramData.bucketCount,
    );
    // Initialize windows using a loop for performance
    for (let i = 0; i < histogramData.bucketCount; i++) {
      windows[i] = { all: {}, highlighted: {} } as HistogramBucketViewModel;
      for (const severity of allLogsHistogramCache.severities) {
        windows[i].all[severity.id] =
          histogramData.logRatios[severity.id][i] /
          histogramData.maxBucketSumRatio;
        windows[i].highlighted[severity.id] =
          filteredHistogramData.logRatios[severity.id][i] /
          histogramData.maxBucketSumRatio;
      }
    }

    const timeLabels = this.generateTimeLabels(
      leftEdgeTimeNs,
      tickTimeNs,
      pixelsPerNs,
      timezoneShiftHours,
      ticks,
      step.minimumTimeLabelSpaceInPx,
      step.labelConverter,
    );

    return {
      tickTimeNs: tickTimeNs,
      histogramBucketTimeNs: histogramData.bucketTimeNs,
      histogramBeginTimeNs: histogramData.histogramBeginTimeNs,
      ticks,
      histogramBuckets: windows,
      timeLabels,
    };
  }

  /**
   * Generates time labels for the ruler, ensuring they don't overlap.
   */
  private generateTimeLabels(
    leftEdgeTimeNs: bigint,
    windowTimeNs: bigint,
    pixelsPerNs: number,
    timezoneShiftHours: number,
    ticks: RulerTickViewModel[],
    minimumTimeLabelSpaceInPx: number,
    labelConverter: NsTimeToLabelConverter,
  ): RulerTimeLabelViewModel[] {
    let labels: RulerTimeLabelViewModel[] = [];
    const labelIndices: bigint[] = [];
    let currentTimeNs = leftEdgeTimeNs;
    let edgeIndex = floorDiv(leftEdgeTimeNs, windowTimeNs);
    const timezoneShiftInNs = BigInt(
      Math.round(timezoneShiftHours * 60 * 60 * 1_000_000_000),
    );
    for (const tick of ticks) {
      if (tick.leftEdgeTimeImportance === TickImportance.High) {
        labels.push({
          offsetLeftInPx: Number(currentTimeNs - leftEdgeTimeNs) * pixelsPerNs,
          label: labelConverter(edgeIndex * windowTimeNs + timezoneShiftInNs),
        });
        labelIndices.push(edgeIndex);
      }
      currentTimeNs += windowTimeNs;
      edgeIndex++;
    }
    const renderAreaWidth = ticks.length * Number(windowTimeNs) * pixelsPerNs;
    const maxLabelCount = Math.ceil(
      renderAreaWidth / minimumTimeLabelSpaceInPx,
    );
    const decimateRatio = BigInt(
      Math.max(1, Math.ceil(Math.log2(labels.length / maxLabelCount))),
    ); // 2^decimateRatio is the ratio to decimate
    if (labels.length > 2) {
      const stride = labelIndices[1] - labelIndices[0];
      labels = labels.filter(
        (_, index) => labelIndices[index] % (stride * decimateRatio) === 0n,
      );
    }
    return labels;
  }
}
