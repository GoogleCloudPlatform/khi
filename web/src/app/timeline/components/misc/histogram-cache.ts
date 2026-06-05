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

import { Severity } from 'src/app/store/domain/style';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Result of the histogram calculation.
 */
export interface HistogramInfo {
  /**
   * The ratio of logs for each severity in the bucket.
   * The key is the severity ID, and the value is an array of ratios (0-1).
   */
  logRatios: { [severityId: number]: Float32Array };
  /**
   * The number of buckets in the result.
   */
  bucketCount: number;
  /**
   * The duration of each window in milliseconds.
   */
  bucketTimeMs: number;

  /**
   * The total number of logs used for calculating the ratio.
   */
  totalLogCount: number;

  /**
   * The maximum sum of ratios for any window.
   */
  maxBucketSumRatio: number;

  /**
   * The start time of the histogram and its the left most time of the first window.
   */
  histogramBeginTimeMs: number;
}

/**
 * Data structure for caching histogram data to optimize rendering performance.
 *
 * It aggregates cumulative counts of logs for each {@link Severity} at a granularity of `minBucketTime`.
 * When querying for histogram data with a specific `windowTimeMS` (which should be an integer multiple of `minBucketTime`),
 * it efficiently calculates the log counts for that window width using the pre-calculated cumulative sums.
 */
export class HistogramCache {
  /**
   * Cumulative sum of log counts for each severity.
   * `cumulativeSums[severity][i]` stores the total number of logs with that severity
   * from `alignedMinTimeMS` up to the end of the i-th timer tick.
   * Used to calculate the number of logs in any time window in O(1) time.
   */
  private readonly cumulativeSums: { [severityId: number]: Int32Array };

  /**
   * Cache for the result of {@link getHistogramData} to avoid allocating new arrays every time.
   */
  private readonly resultCache: { [severityId: number]: Float32Array };

  /**
   * The start time (timestamp in milliseconds) of the histogram cache, aligned to `minBucketTimeMS`.
   * This corresponds to index 0 in the `cumulativeSums` arrays.
   */
  private readonly alignedMinTimeMS: number;

  /**
   * Creates a new HistogramCache.
   *
   * @param severities - The list of severities.
   * @param logs - The list of logs to be indexed.
   * @param minBucketTime - The resolution of the cache in milliseconds.
   * @param logMinTimeMS - The minimum time of the logs. It will be recalculated from given logs, but this allows user to extend the range.
   * @param logMaxTimeMS - The maximum time of the logs. It will be recalculated from given logs, but this allows user to extend the range.
   */
  constructor(
    public readonly severities: readonly ReadonlyDomainElement<Severity>[],
    logs: readonly ReadonlyDomainElement<Log>[],
    private readonly minBucketTime: number,
    public logMinTimeMS: number = Infinity,
    public logMaxTimeMS: number = -Infinity,
  ) {
    if (logs.length === 0) {
      this.alignedMinTimeMS = 0;
      this.cumulativeSums = {} as { [severityId: number]: Int32Array };
      this.resultCache = {} as { [severityId: number]: Float32Array };
      for (const severity of this.severities) {
        this.cumulativeSums[severity.id] = new Int32Array(0);
        this.resultCache[severity.id] = new Float32Array(0);
      }
      return;
    }

    for (const log of logs) {
      this.logMinTimeMS = Math.min(this.logMinTimeMS, log.legacyTimestampMs);
      this.logMaxTimeMS = Math.max(this.logMaxTimeMS, log.legacyTimestampMs);
    }
    this.alignedMinTimeMS =
      Math.floor(this.logMinTimeMS / minBucketTime) * minBucketTime;
    const timeAlignedMaxTime =
      Math.ceil(this.logMaxTimeMS / minBucketTime) * minBucketTime;
    const windowCount =
      Math.ceil((timeAlignedMaxTime - this.alignedMinTimeMS) / minBucketTime) +
      1;

    this.cumulativeSums = {} as { [severityId: number]: Int32Array };
    this.resultCache = {} as { [severityId: number]: Float32Array };

    for (const severity of this.severities) {
      this.cumulativeSums[severity.id] = new Int32Array(windowCount);
      this.resultCache[severity.id] = new Float32Array(windowCount);
    }
    for (const log of logs) {
      const windowIndex =
        (Math.floor(log.legacyTimestampMs / minBucketTime) * minBucketTime -
          this.alignedMinTimeMS) /
          minBucketTime +
        1;
      if (windowIndex >= 0 && windowIndex < windowCount) {
        this.cumulativeSums[log.severity.id][windowIndex]++;
      }
    }
    // Calculate the cumulative values
    for (let i = 1; i < windowCount; i++) {
      for (const severity of this.severities) {
        this.cumulativeSums[severity.id][i] +=
          this.cumulativeSums[severity.id][i - 1];
      }
    }
  }

  /**
   * Retrieves histogram data for the specified time range and window size.
   *
   * @param timeAlignedMinTimeMS - The start time of the range, aligned to the tick time.
   * @param timeAlignedMaxTimeMS - The end time of the range, aligned to the tick time.
   * @param bucketTimeMs - The duration of each window in milliseconds. Must be equal to or greater than `minBucketTimeMS`.
   * @returns The histogram information containing log ratios per severity.
   */
  public getHistogramData(
    timeAlignedMinTimeMS: number,
    timeAlignedMaxTimeMS: number,
    bucketTimeMs: number,
    totalLogCount?: number,
  ): HistogramInfo {
    bucketTimeMs = Math.max(bucketTimeMs, this.minBucketTime);
    const firstSeverityId = this.severities[0]?.id;
    if (
      firstSeverityId === undefined ||
      this.cumulativeSums[firstSeverityId].length === 0
    ) {
      return {
        logRatios: this.resultCache,
        bucketCount: 0,
        bucketTimeMs: bucketTimeMs,
        totalLogCount: 0,
        maxBucketSumRatio: 0,
        histogramBeginTimeMs: 0,
      };
    }
    const windowStride = Math.round(bucketTimeMs / this.minBucketTime);
    const logRatios: { [severityId: number]: Float32Array } = this.resultCache;
    const histogramBeginTimeMs =
      Math.floor(timeAlignedMinTimeMS / bucketTimeMs) * bucketTimeMs;
    const histogramEndTimeMs =
      Math.ceil(timeAlignedMaxTimeMS / bucketTimeMs) * bucketTimeMs;
    const leftMostTimeIndex = Math.round(
      (histogramBeginTimeMs - this.alignedMinTimeMS) / this.minBucketTime,
    );
    const rightMostTimeIndex = Math.round(
      (histogramEndTimeMs - this.alignedMinTimeMS) / this.minBucketTime,
    );

    // When totalLogCount is not provided, calculate it from the logs array.
    if (totalLogCount === undefined) {
      totalLogCount = 0;
      for (const severity of this.severities) {
        totalLogCount += this.logCountForSeverity(
          severity.id,
          histogramBeginTimeMs,
          histogramEndTimeMs,
        );
      }
    }

    for (const severity of this.severities) {
      let currentResultIndex = 0;
      for (
        let currentTimeIndex = leftMostTimeIndex;
        currentTimeIndex < rightMostTimeIndex;
        currentTimeIndex += windowStride
      ) {
        const beginWindowTime =
          currentTimeIndex * this.minBucketTime + this.alignedMinTimeMS;
        const endWindowTime =
          (currentTimeIndex + windowStride) * this.minBucketTime +
          this.alignedMinTimeMS;
        if (totalLogCount > 0) {
          logRatios[severity.id][currentResultIndex] =
            this.logCountForSeverity(
              severity.id,
              beginWindowTime,
              endWindowTime,
            ) / totalLogCount;
        } else {
          logRatios[severity.id][currentResultIndex] = 0;
        }
        currentResultIndex++;
      }
    }
    const bucketCount = Math.round(
      (histogramEndTimeMs - histogramBeginTimeMs) / bucketTimeMs,
    );
    let maxBucketSumRatio = -Infinity;
    for (let i = 0; i < bucketCount; i++) {
      let sumRatio = 0;
      for (const severity of this.severities) {
        sumRatio += logRatios[severity.id][i];
      }
      maxBucketSumRatio = Math.max(maxBucketSumRatio, sumRatio);
    }
    return {
      logRatios,
      bucketCount,
      bucketTimeMs,
      totalLogCount,
      maxBucketSumRatio,
      histogramBeginTimeMs,
    };
  }

  /**
   * Calculates the number of logs with the specified severity within the given time range.
   * This method uses the pre-calculated `cumulativeSums` to perform the calculation in O(1) time.
   *
   * @param severityId - The severity ID of the logs to count.
   * @param alignedLeftTimeMS - The start time of the range (inclusive), aligned to `minTickTimeMS`.
   * @param alignedRightTimeMS - The end time of the range (exclusive), aligned to `minTickTimeMS`.
   * @returns The number of logs in the specified range.
   */
  private logCountForSeverity(
    severityId: number,
    alignedLeftTimeMS: number,
    alignedRightTimeMS: number,
  ): number {
    const minIndex = Math.round(
      (alignedLeftTimeMS - this.alignedMinTimeMS) / this.minBucketTime,
    );
    const maxIndex = Math.round(
      (alignedRightTimeMS - this.alignedMinTimeMS) / this.minBucketTime,
    );
    let maxValue =
      this.cumulativeSums[severityId][
        this.cumulativeSums[severityId].length - 1
      ];
    let minValue = 0;
    if (minIndex >= 0) {
      if (minIndex < this.cumulativeSums[severityId].length) {
        minValue = this.cumulativeSums[severityId][minIndex];
      } else {
        minValue =
          this.cumulativeSums[severityId][
            this.cumulativeSums[severityId].length - 1
          ];
      }
    }
    if (maxIndex < this.cumulativeSums[severityId].length) {
      if (maxIndex >= 0) {
        maxValue = this.cumulativeSums[severityId][maxIndex];
      } else {
        maxValue = 0;
      }
    }

    return Math.max(0, maxValue - minValue);
  }
}
