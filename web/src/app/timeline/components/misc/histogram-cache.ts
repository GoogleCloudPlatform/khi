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
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';

function floorDiv(a: bigint, b: bigint): bigint {
  const q = a / b;
  const r = a % b;
  return r !== 0n && a < 0n !== b < 0n ? q - 1n : q;
}

function ceilDiv(a: bigint, b: bigint): bigint {
  const q = a / b;
  const r = a % b;
  return r !== 0n && a < 0n === b < 0n ? q + 1n : q;
}

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
   * The duration of each window in nanoseconds.
   */
  bucketTimeNs: bigint;

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
  histogramBeginTimeNs: bigint;
}

/**
 * Data structure for caching histogram data to optimize rendering performance.
 *
 * It aggregates cumulative counts of logs for each {@link Severity} at a granularity of `minBucketTimeNs`.
 * When querying for histogram data with a specific `bucketTimeNs` (which should be an integer multiple of `minBucketTimeNs`),
 * it efficiently calculates the log counts for that window width using the pre-calculated cumulative sums.
 */
export class HistogramCache {
  /**
   * Cumulative sum of log counts for each severity.
   * `cumulativeSums[severity][i]` stores the total number of logs with that severity
   * from `alignedMinTimeNs` up to the end of the i-th timer tick.
   * Used to calculate the number of logs in any time window in O(1) time.
   */
  private readonly cumulativeSums: { [severityId: number]: Int32Array };

  /**
   * Cache for the result of {@link getHistogramData} to avoid allocating new arrays every time.
   */
  private readonly resultCache: { [severityId: number]: Float32Array };

  /**
   * The start time (timestamp in nanoseconds) of the histogram cache, aligned to `minBucketTimeNs`.
   * This corresponds to index 0 in the `cumulativeSums` arrays.
   */
  private readonly alignedMinTimeNs: bigint;

  /**
   * Creates a new HistogramCache.
   *
   * The range is given by the caller rather than derived from the logs because
   * `minBucketTimeNs` is chosen from the same range. Deriving the range from the logs
   * lets a single log far outside the range inflate the bucket count without bound.
   * Logs outside the range are not counted.
   *
   * @param severities - The list of severities.
   * @param logs - The list of logs to be indexed.
   * @param logIds - The bitset of active log IDs to include in the histogram.
   * @param minBucketTimeNs - The resolution of the cache in nanoseconds.
   * @param rangeBeginTimeNs - The start time of the histogram range in nanoseconds.
   * @param rangeEndTimeNs - The end time of the histogram range in nanoseconds. The bucket containing this time is included.
   */
  constructor(
    public readonly severities: readonly ReadonlyDomainElement<Severity>[],
    logs: readonly ReadonlyDomainElement<Log>[],
    logIds: IdBitset,
    private readonly minBucketTimeNs: bigint,
    rangeBeginTimeNs: bigint,
    rangeEndTimeNs: bigint,
  ) {
    if (logs.length === 0 || logIds.size === 0) {
      this.alignedMinTimeNs = 0n;
      this.cumulativeSums = {} as { [severityId: number]: Int32Array };
      this.resultCache = {} as { [severityId: number]: Float32Array };
      for (const severity of this.severities) {
        this.cumulativeSums[severity.id] = new Int32Array(0);
        this.resultCache[severity.id] = new Float32Array(0);
      }
      return;
    }

    this.alignedMinTimeNs =
      floorDiv(rangeBeginTimeNs, minBucketTimeNs) * minBucketTimeNs;
    const alignedMaxTimeNs =
      (floorDiv(rangeEndTimeNs, minBucketTimeNs) + 1n) * minBucketTimeNs;
    const windowCount =
      Number((alignedMaxTimeNs - this.alignedMinTimeNs) / minBucketTimeNs) + 1;

    this.cumulativeSums = {} as { [severityId: number]: Int32Array };
    this.resultCache = {} as { [severityId: number]: Float32Array };

    for (const severity of this.severities) {
      this.cumulativeSums[severity.id] = new Int32Array(windowCount);
      this.resultCache[severity.id] = new Float32Array(windowCount);
    }
    for (const log of logs) {
      if (!logIds.has(log.id)) {
        continue;
      }
      const windowIndex =
        Number(
          (floorDiv(log.timestamp, minBucketTimeNs) * minBucketTimeNs -
            this.alignedMinTimeNs) /
            minBucketTimeNs,
        ) + 1;
      // Index 0 is the empty prefix of the cumulative sums, so logs before the range (index <= 0) and after it are skipped.
      if (windowIndex > 0 && windowIndex < windowCount) {
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
   * @param timeAlignedMinTimeNs - The start time of the range, aligned to the tick time.
   * @param timeAlignedMaxTimeNs - The end time of the range, aligned to the tick time.
   * @param bucketTimeNs - The duration of each window in nanoseconds. Must be equal to or greater than `minBucketTimeNs`.
   * @returns The histogram information containing log ratios per severity.
   */
  public getHistogramData(
    timeAlignedMinTimeNs: bigint,
    timeAlignedMaxTimeNs: bigint,
    bucketTimeNs: bigint,
    totalLogCount?: number,
  ): HistogramInfo {
    if (bucketTimeNs < this.minBucketTimeNs) {
      bucketTimeNs = this.minBucketTimeNs;
    }
    const firstSeverityId = this.severities[0]?.id;
    if (
      firstSeverityId === undefined ||
      this.cumulativeSums[firstSeverityId].length === 0
    ) {
      return {
        logRatios: this.resultCache,
        bucketCount: 0,
        bucketTimeNs: bucketTimeNs,
        totalLogCount: 0,
        maxBucketSumRatio: 0,
        histogramBeginTimeNs: 0n,
      };
    }
    const windowStride = Number(bucketTimeNs / this.minBucketTimeNs);
    const logRatios: { [severityId: number]: Float32Array } = this.resultCache;
    const histogramBeginTimeNs =
      floorDiv(timeAlignedMinTimeNs, bucketTimeNs) * bucketTimeNs;
    const histogramEndTimeNs =
      ceilDiv(timeAlignedMaxTimeNs, bucketTimeNs) * bucketTimeNs;
    const leftMostTimeIndex = Number(
      (histogramBeginTimeNs - this.alignedMinTimeNs) / this.minBucketTimeNs,
    );
    const rightMostTimeIndex = Number(
      (histogramEndTimeNs - this.alignedMinTimeNs) / this.minBucketTimeNs,
    );

    // When totalLogCount is not provided, calculate it from the logs array.
    if (totalLogCount === undefined) {
      totalLogCount = 0;
      for (const severity of this.severities) {
        totalLogCount += this.logCountForSeverity(
          severity.id,
          histogramBeginTimeNs,
          histogramEndTimeNs,
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
        const beginWindowTimeNs =
          BigInt(currentTimeIndex) * this.minBucketTimeNs +
          this.alignedMinTimeNs;
        const endWindowTimeNs =
          BigInt(currentTimeIndex + windowStride) * this.minBucketTimeNs +
          this.alignedMinTimeNs;
        if (totalLogCount > 0) {
          logRatios[severity.id][currentResultIndex] =
            this.logCountForSeverity(
              severity.id,
              beginWindowTimeNs,
              endWindowTimeNs,
            ) / totalLogCount;
        } else {
          logRatios[severity.id][currentResultIndex] = 0;
        }
        currentResultIndex++;
      }
    }
    const bucketCount = Number(
      (histogramEndTimeNs - histogramBeginTimeNs) / bucketTimeNs,
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
      bucketTimeNs,
      totalLogCount,
      maxBucketSumRatio,
      histogramBeginTimeNs,
    };
  }

  /**
   * Calculates the number of logs with the specified severity within the given time range.
   * This method uses the pre-calculated `cumulativeSums` to perform the calculation in O(1) time.
   *
   * @param severityId - The severity ID of the logs to count.
   * @param alignedLeftTimeNs - The start time of the range (inclusive), aligned to `minBucketTimeNs`.
   * @param alignedRightTimeNs - The end time of the range (exclusive), aligned to `minBucketTimeNs`.
   * @returns The number of logs in the specified range.
   */
  private logCountForSeverity(
    severityId: number,
    alignedLeftTimeNs: bigint,
    alignedRightTimeNs: bigint,
  ): number {
    const minIndex = Number(
      (alignedLeftTimeNs - this.alignedMinTimeNs) / this.minBucketTimeNs,
    );
    const maxIndex = Number(
      (alignedRightTimeNs - this.alignedMinTimeNs) / this.minBucketTimeNs,
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
