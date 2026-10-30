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

import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { HistogramCache } from './histogram-cache';

enum Severity {
  SeverityUnknown = 0,
  SeverityInfo = 1,
  SeverityWarning = 2,
  SeverityError = 3,
  SeverityFatal = 4,
}

function createCacheWithLogs(
  severityTimes: { severity: Severity; timeNs: bigint }[],
  minBucketTimeNs: bigint,
  rangeBeginTimeNs: bigint,
  rangeEndTimeNs: bigint,
  filterLogIds?: IdBitset,
): HistogramCache {
  const internPool = InternPoolStore.create();
  const styleStore = new StyleStore();

  styleStore.addSeverities([
    {
      id: Severity.SeverityInfo,
      label: 'Info',
      shortLabel: 'I',
      backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      order: 1,
    },
    {
      id: Severity.SeverityError,
      label: 'Error',
      shortLabel: 'E',
      backgroundColor: { r: 1, g: 0, b: 0, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      order: 2,
    },
  ]);

  const logsDto = severityTimes.map((item, index) => ({
    id: index + 1,
    ts: item.timeNs,
    logTypeId: 0,
    severityTypeId: item.severity,
    summaryStringId: 0,
  }));

  const logStore = LogStore.create(internPool, styleStore, logsDto.length);
  logStore.addLogs(logsDto);
  logStore.shrinkToFit();
  const logs = Array.from(logStore.logs());
  const logIds = filterLogIds ?? IdBitset.fromSequential(logs.length);
  return new HistogramCache(
    styleStore.severities,
    logs,
    logIds,
    minBucketTimeNs,
    rangeBeginTimeNs,
    rangeEndTimeNs,
  );
}

describe('HistogramCache', () => {
  it('should correctly calculate ratios for a basic scenario', () => {
    // 1000n ticks, Range: 0n - 3000n
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeNs: 500n }, // Window 0 (0-1000)
        { severity: Severity.SeverityError, timeNs: 1500n }, // Window 1 (1000-2000)
        { severity: Severity.SeverityInfo, timeNs: 2500n }, // Window 2 (2000-3000)
      ],
      1000n,
      0n,
      3000n,
    );

    // Get data for 0-3000 with window 1000
    const result = cache.getHistogramData(0n, 3000n, 1000n);

    // Total Logs: 3 (2 Errors + 1 Info)

    // Error: Total 2.
    // Window 0 (0-1000): 1 log -> 1/3 ratio
    // Window 1 (1000-2000): 1 log -> 1/3 ratio
    // Window 2 (2000-3000): 0 log -> 0 ratio
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(1 / 3);
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(1 / 3);
    expect(result.logRatios[Severity.SeverityError][2]).toBeCloseTo(0);

    // Info: Total 1.
    // Window 0 (0-1000): 0 log -> 0 ratio
    // Window 1 (1000-2000): 0 log -> 0 ratio
    // Window 2 (2000-3000): 1 log -> 1/3 ratio
    expect(result.logRatios[Severity.SeverityInfo][0]).toBeCloseTo(0);
    expect(result.logRatios[Severity.SeverityInfo][1]).toBeCloseTo(0);
    expect(result.logRatios[Severity.SeverityInfo][2]).toBeCloseTo(1 / 3);
    // Total windows should be 3 (3000 / 1000)
    expect(result.bucketCount).toBe(3);
  });

  it('should handle aggregated windows', () => {
    // Min tick 1000n, Window 2000n
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeNs: 500n }, // Window 0 (0-2000)
        { severity: Severity.SeverityError, timeNs: 1500n }, // Window 0 (0-2000)
        { severity: Severity.SeverityError, timeNs: 2500n }, // Window 1 (2000-4000)
      ],
      1000n,
      0n,
      4000n,
    );

    const result = cache.getHistogramData(0n, 4000n, 2000n);

    // Total Error logs: 3
    // Window 0 (0-2000): 2 logs -> 2/3
    // Window 1 (2000-4000): 1 log -> 1/3
    // Indices in result array will be 0 and 2 because stride is 2 (2000/1000)
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(2 / 3);
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(1 / 3);

    // Total windows should be 2 (4000 / 2000)
    expect(result.bucketCount).toBe(2);
  });

  it('should handle no logs gracefully', () => {
    const cache = createCacheWithLogs([], 1000n, 0n, 3000n);
    const result = cache.getHistogramData(0n, 3000n, 1000n);

    expect(result.logRatios[Severity.SeverityError].length).toBe(0);
    expect(result.bucketCount).toBe(0);
  });

  it('should handle non-zero start time correctly', () => {
    // 1000n ticks. The cache range is 0-4000 and the query below starts at 2000.
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeNs: 500n },
        { severity: Severity.SeverityError, timeNs: 1500n },
        { severity: Severity.SeverityInfo, timeNs: 2500n },
        { severity: Severity.SeverityError, timeNs: 3500n },
      ],
      1000n,
      0n,
      4000n,
    );

    // Total logs in [2000, 4000): 2500 (Info), 3500 (Error). Total 2.
    const result = cache.getHistogramData(2000n, 4000n, 1000n);

    // Index 0 (2000-3000): Info (1). Ratio 1/2.
    expect(result.logRatios[Severity.SeverityInfo][0]).toBeCloseTo(0.5);
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(0);

    // Index 1 (3000-4000): Error (1). Ratio 1/2.
    expect(result.logRatios[Severity.SeverityInfo][1]).toBeCloseTo(0);
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(0.5);
  });

  it('should treat bucketTimeNs smaller than minBucketTimeNs as minBucketTimeNs', () => {
    // 1000n ticks.
    const cache = createCacheWithLogs(
      [{ severity: Severity.SeverityError, timeNs: 500n }],
      1000n,
      0n,
      1000n,
    );

    // Request window 500n (< 1000n)
    // Should behave as 1000n window.
    const result = cache.getHistogramData(0n, 1000n, 500n);

    // Window 0 (0-1000): 1 log. Total 1. Ratio 1.0.
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(1.0);
    expect(result.bucketCount).toBe(1);
  });

  it('should only include logs present in the logIds bitset', () => {
    // 3 logs with IDs 1, 2, 3
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeNs: 500n }, // ID 1
        { severity: Severity.SeverityError, timeNs: 1500n }, // ID 2
        { severity: Severity.SeverityInfo, timeNs: 2500n }, // ID 3
      ],
      1000n,
      0n,
      3000n,
      IdBitset.fromAll([1, 3]), // Include only ID 1 and ID 3
    );

    const result = cache.getHistogramData(0n, 3000n, 1000n);

    // Total filtered logs: 2 (ID 1 Error, ID 3 Info)
    expect(result.totalLogCount).toBe(2);
    // Window 0: 1 Error -> 1/2
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(0.5);
    // Window 1 (ID 2 omitted): 0 -> 0
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(0);
    // Window 2: 1 Info -> 1/2
    expect(result.logRatios[Severity.SeverityInfo][2]).toBeCloseTo(0.5);
  });

  it('should ignore logs outside the range without growing the cache', () => {
    const baseNs = 1_700_000_000_000_000_000n;
    const hourNs = 60n * 60n * 1_000_000_000n;
    const yearNs = 365n * 24n * hourNs;
    const bucketNs = 500_000_000n;
    const cache = createCacheWithLogs(
      [
        // Years before the range, like a snapshot log stamped with an old change time.
        { severity: Severity.SeverityError, timeNs: baseNs - 8n * yearNs },
        // Within one bucket before the range start.
        { severity: Severity.SeverityError, timeNs: baseNs - 250_000_000n },
        { severity: Severity.SeverityError, timeNs: baseNs + 1_000_000_000n },
        { severity: Severity.SeverityInfo, timeNs: baseNs + 2_000_000_000n },
        // After the range end.
        { severity: Severity.SeverityError, timeNs: baseNs + 2n * hourNs },
      ],
      bucketNs,
      baseNs,
      baseNs + hourNs,
    );

    // Start one bucket before the range so that logs counted before the range would be included.
    const result = cache.getHistogramData(
      baseNs - bucketNs,
      baseNs + hourNs,
      bucketNs,
    );

    expect(result.totalLogCount).toBe(2);
    // One prefix slot plus the buckets covering the range, including the bucket containing the range end.
    expect(result.logRatios[Severity.SeverityError].length).toBe(
      Number(hourNs / bucketNs) + 2,
    );
  });

  it('should count a log exactly at the range end', () => {
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeNs: 500n },
        { severity: Severity.SeverityError, timeNs: 3000n },
      ],
      1000n,
      0n,
      3000n,
    );

    const result = cache.getHistogramData(0n, 4000n, 1000n);

    expect(result.totalLogCount).toBe(2);
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(0.5);
    expect(result.logRatios[Severity.SeverityError][3]).toBeCloseTo(0.5);
  });
});
