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

import { Severity } from 'src/app/generated';
import { HistogramCache } from './misc/histogram-cache';
import {
  getRulerStep,
  MSTimeToLabelConverter,
} from './calculator/human-friendly-tick';

/**
 * The view model for the timeline ruler(header part of the timeline component).
 */
export interface TimelineRulerViewModel {
  /**
   * The time duration of the smallest scale step.
   */
  tickTimeMS: number;

  /**
   * The time duration of a histogram bucket.
   * This must be divisible by tickTimeMS to align the historam and scales.
   */
  histogramBucketTimeMS: number;

  /**
   * The start time of the histogram.
   */
  histogramBeginTimeMS: number;

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
  all: { [severity in Severity]: number };

  /**
   * [0-1] ratio of the highlighted log count in the bucket for each severity.
   */
  highlighted: { [severity in Severity]: number };
}

/**
 * The importance of a tick. This is used to determine the style of the tick rendered on the header.
 */
export enum TickImportance {
  Low,
  Middle,
  High,
}

const severities = Object.values(Severity).filter(
  (s) => !isNaN(Number(s)),
) as Severity[];

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
   * @param leftEdgeTimeMS The time at the left edge of the viewport (ms).
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @param viewportWidth The width of the viewport (px).
   * @param timezoneShiftHours The timezone offset in hours to adjust time labels.
   * @param allLogsHistogramCache Cache for all logs histogram data.
   * @param filteredLogsHistogramCache Cache for filtered logs histogram data.
   * @returns The generated TimelineRulerViewModel.
   */
  generateRulerViewModel(
    leftEdgeTimeMS: number,
    pixelsPerMs: number,
    viewportWidth: number,
    timezoneShiftHours: number,
    allLogsHistogramCache: HistogramCache,
    filteredLogsHistogramCache: HistogramCache,
  ): TimelineRulerViewModel {
    const step = getRulerStep(pixelsPerMs, this.minTickWidthInPx);

    const tickTimeMS = step.low;
    const startTickIndex = Math.floor(leftEdgeTimeMS / tickTimeMS);

    const numberOfTicks =
      Math.ceil(
        (viewportWidth + this.extraOffsetWidthInPx * 2) /
          (tickTimeMS * pixelsPerMs),
      ) + 1; // +1 for buffer
    const ticks: RulerTickViewModel[] = new Array(numberOfTicks);
    for (let i = 0; i < numberOfTicks; i++) {
      const currentTickIndex = startTickIndex + i;
      let importance = TickImportance.Low;
      if (currentTickIndex % step.highMultiplier === 0) {
        importance = TickImportance.High;
      } else if (currentTickIndex % step.middleMultiplier === 0) {
        importance = TickImportance.Middle;
      }
      ticks[i] = {
        leftEdgeTimeImportance: importance,
      };
    }

    const histogramData = allLogsHistogramCache.getHistogramData(
      startTickIndex * tickTimeMS,
      (startTickIndex + numberOfTicks) * tickTimeMS,
      tickTimeMS,
    );
    const filteredHistogramData = filteredLogsHistogramCache.getHistogramData(
      startTickIndex * tickTimeMS,
      (startTickIndex + numberOfTicks) * tickTimeMS,
      tickTimeMS,
      histogramData.totalLogCount,
    );
    const windows: HistogramBucketViewModel[] = new Array(
      histogramData.bucketCount,
    );
    // Initialize windows using a loop for performance
    for (let i = 0; i < histogramData.bucketCount; i++) {
      windows[i] = { all: {}, highlighted: {} } as HistogramBucketViewModel;
      for (const severity of severities) {
        windows[i].all[severity] =
          histogramData.logRatios[severity][i] /
          histogramData.maxBucketSumRatio;
        windows[i].highlighted[severity] =
          filteredHistogramData.logRatios[severity][i] /
          histogramData.maxBucketSumRatio;
      }
    }

    const timeLabels = this.generateTimeLabels(
      leftEdgeTimeMS,
      tickTimeMS,
      pixelsPerMs,
      timezoneShiftHours,
      ticks,
      step.minimumTimeLabelSpaceInPx,
      step.labelConverter,
    );

    return {
      tickTimeMS: tickTimeMS,
      histogramBucketTimeMS: histogramData.bucketTimeMs,
      histogramBeginTimeMS: histogramData.histogramBeginTimeMs,
      ticks,
      histogramBuckets: windows,
      timeLabels,
    };
  }

  /**
   * Generates time labels for the ruler, ensuring they don't overlap.
   */
  private generateTimeLabels(
    leftEdgeTimeMS: number,
    windowTimeMS: number,
    pixelsPerMs: number,
    timezoneShiftHours: number,
    ticks: RulerTickViewModel[],
    minimumTimeLabelSpaceInPx: number,
    labelConverter: MSTimeToLabelConverter,
  ): RulerTimeLabelViewModel[] {
    let labels: RulerTimeLabelViewModel[] = [];
    const labelIndices: number[] = [];
    let currentTime = leftEdgeTimeMS;
    let edgeIndex = Math.floor(leftEdgeTimeMS / windowTimeMS);
    for (const tick of ticks) {
      if (tick.leftEdgeTimeImportance === TickImportance.High) {
        labels.push({
          offsetLeftInPx: (currentTime - leftEdgeTimeMS) * pixelsPerMs,
          label: labelConverter(
            edgeIndex * windowTimeMS + timezoneShiftHours * 60 * 60 * 1000,
          ),
        });
        labelIndices.push(edgeIndex);
      }
      currentTime += windowTimeMS;
      edgeIndex++;
    }
    const renderAreaWidth = ticks.length * windowTimeMS * pixelsPerMs;
    const maxLabelCount = Math.ceil(
      renderAreaWidth / minimumTimeLabelSpaceInPx,
    );
    const decimateRatio = Math.max(
      1,
      Math.ceil(Math.log2(labels.length / maxLabelCount)),
    ); // 2^decimateRatio is the ratio to decimate
    if (labels.length > 2) {
      const stride = labelIndices[1] - labelIndices[0];
      labels = labels.filter(
        (_, index) => labelIndices[index] % (stride * decimateRatio) === 0,
      );
    }
    return labels;
  }
}
