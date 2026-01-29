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

import { Severity } from 'src/app/zzz-generated';
import { LogEntry } from 'src/app/store/log';

/**
 * Result of the histogram calculation.
 */
export interface HistogramInfo {
  /**
   * The ratio of logs for each severity in the bucket.
   * The key is the severity, and the value is an array of ratios (0-1).
   */
  logRatios: { [key in Severity]: Float32Array };
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

// The list of all severity keys.
const severities = Object.entries(Severity)
  .filter(([, value]) => !isNaN(Number(value)))
  .map(([, value]) => value) as Severity[];

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
  private readonly cumulativeSums: { [key in Severity]: Int32Array };

  /**
   * Cache for the result of {@link getHistogramData} to avoid allocating new arrays every time.
   */
  private readonly resultCache: { [key in Severity]: Float32Array };

  /**
   * The start time (timestamp in milliseconds) of the histogram cache, aligned to `minBucketTimeMS`.
   * This corresponds to index 0 in the `cumulativeSums` arrays.
   */
  private readonly alignedMinTimeMS: number;

  /**
   * Creates a new HistogramCache.
   *
   * @param logs - The list of logs to be indexed.
   * @param minBucketTime - The resolution of the cache in milliseconds.
   * @param logMinTimeMS - The minimum time of the logs. It will be recalculated from given logs, but this allows user to extend the range.
   * @param logMaxTimeMS - The maximum time of the logs. It will be recalculated from given logs, but this allows user to extend the range.
   */
  constructor(
    logs: LogEntry[],
    private readonly minBucketTime: number,
    public logMinTimeMS: number = Infinity,
    public logMaxTimeMS: number = -Infinity,
  ) {
    if (logs.length === 0) {
      this.alignedMinTimeMS = 0;
      this.cumulativeSums = {} as { [key in Severity]: Int32Array };
      this.resultCache = {} as { [key in Severity]: Float32Array };
      for (const severity of severities) {
        this.cumulativeSums[severity] = new Int32Array(0);
        this.resultCache[severity] = new Float32Array(0);
      }
      return;
    }

    for (const log of logs) {
      this.logMinTimeMS = Math.min(this.logMinTimeMS, log.time);
      this.logMaxTimeMS = Math.max(this.logMaxTimeMS, log.time);
    }
    this.alignedMinTimeMS =
      Math.floor(this.logMinTimeMS / minBucketTime) * minBucketTime;
    const timeAlignedMaxTime =
      Math.ceil(this.logMaxTimeMS / minBucketTime) * minBucketTime;
    const windowCount =
      Math.ceil((timeAlignedMaxTime - this.alignedMinTimeMS) / minBucketTime) +
      1;

    this.cumulativeSums = {} as { [key in Severity]: Int32Array };
    this.resultCache = {} as { [key in Severity]: Float32Array };

    for (const severity of severities) {
      this.cumulativeSums[severity] = new Int32Array(windowCount);
      this.resultCache[severity] = new Float32Array(windowCount);
    }
    for (const log of logs) {
      const windowIndex =
        (Math.floor(log.time / minBucketTime) * minBucketTime -
          this.alignedMinTimeMS) /
          minBucketTime +
        1;
      if (windowIndex >= 0 && windowIndex < windowCount) {
        this.cumulativeSums[log.severity][windowIndex]++;
      }
    }
    // Calculate the cumulative values
    for (let i = 1; i < windowCount; i++) {
      for (const severity of severities) {
        this.cumulativeSums[severity][i] +=
          this.cumulativeSums[severity][i - 1];
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
    if (this.cumulativeSums[Severity.SeverityError].length === 0) {
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
    const logRatios: { [key in Severity]: Float32Array } = this.resultCache;
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
      for (const severity of severities) {
        totalLogCount += this.logCountForSeverity(
          severity,
          histogramBeginTimeMs,
          histogramEndTimeMs,
        );
      }
    }

    for (const severity of severities) {
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
          logRatios[severity][currentResultIndex] =
            this.logCountForSeverity(severity, beginWindowTime, endWindowTime) /
            totalLogCount;
        } else {
          logRatios[severity][currentResultIndex] = 0;
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
      for (const severity of severities) {
        sumRatio += logRatios[severity][i];
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
   * @param severity - The severity of the logs to count.
   * @param alignedLeftTimeMS - The start time of the range (inclusive), aligned to `minTickTimeMS`.
   * @param alignedRightTimeMS - The end time of the range (exclusive), aligned to `minTickTimeMS`.
   * @returns The number of logs in the specified range.
   */
  private logCountForSeverity(
    severity: Severity,
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
      this.cumulativeSums[severity][this.cumulativeSums[severity].length - 1];
    let minValue = 0;
    if (minIndex >= 0) {
      if (minIndex < this.cumulativeSums[severity].length) {
        minValue = this.cumulativeSums[severity][minIndex];
      } else {
        minValue =
          this.cumulativeSums[severity][
            this.cumulativeSums[severity].length - 1
          ];
      }
    }
    if (maxIndex < this.cumulativeSums[severity].length) {
      if (maxIndex >= 0) {
        maxValue = this.cumulativeSums[severity][maxIndex];
      } else {
        maxValue = 0;
      }
    }

    return Math.max(0, maxValue - minValue);
  }
}
