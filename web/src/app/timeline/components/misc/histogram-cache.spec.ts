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
import { HistogramCache } from './histogram-cache';

enum Severity {
  SeverityUnknown = 0,
  SeverityInfo = 1,
  SeverityWarning = 2,
  SeverityError = 3,
  SeverityFatal = 4,
}

function createCacheWithLogs(
  severityTimes: { severity: Severity; timeMs: number }[],
  minBucketTime: number,
  minTimeMs?: number,
  maxTimeMs?: number,
): HistogramCache {
  const internPool = InternPoolStore.create();
  const styleStore = new StyleStore();
  const logStore = LogStore.create(internPool, styleStore);

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
    ts: BigInt(item.timeMs) * 1000000n,
    logTypeId: 0,
    severityTypeId: item.severity,
    summaryStringId: 0,
  }));

  logStore.initialize(logsDto, logsDto.length);
  const logs = Array.from(logStore.logs());
  return new HistogramCache(
    styleStore.severities,
    logs,
    minBucketTime,
    minTimeMs,
    maxTimeMs,
  );
}

describe('HistogramCache', () => {
  it('should correctly calculate ratios for a basic scenario', () => {
    // 1000ms ticks, Range: 0 - 3000ms
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeMs: 500 }, // Window 0 (0-1000)
        { severity: Severity.SeverityError, timeMs: 1500 }, // Window 1 (1000-2000)
        { severity: Severity.SeverityInfo, timeMs: 2500 }, // Window 2 (2000-3000)
      ],
      1000,
    );

    // Get data for 0-3000ms with window 1000ms
    const result = cache.getHistogramData(0, 3000, 1000);

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
    // Total windows should be 3 (3000ms / 1000ms)
    expect(result.bucketCount).toBe(3);
  });

  it('should handle aggregated windows', () => {
    // Min tick 1000ms, Window 2000ms
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeMs: 500 }, // Window 0 (0-2000)
        { severity: Severity.SeverityError, timeMs: 1500 }, // Window 0 (0-2000)
        { severity: Severity.SeverityError, timeMs: 2500 }, // Window 1 (2000-4000)
      ],
      1000,
      0,
      4000,
    );

    const result = cache.getHistogramData(0, 4000, 2000);

    // Total Error logs: 3
    // Window 0 (0-2000): 2 logs -> 2/3
    // Window 1 (2000-4000): 1 log -> 1/3
    // Indices in result array will be 0 and 2 because stride is 2 (2000/1000)
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(2 / 3);
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(1 / 3);

    // Total windows should be 2 (4000ms / 2000ms)
    expect(result.bucketCount).toBe(2);
  });

  it('should handle no logs gracefully', () => {
    const cache = createCacheWithLogs([], 1000);
    const result = cache.getHistogramData(0, 3000, 1000);

    expect(result.logRatios[Severity.SeverityError].length).toBe(0);
    expect(result.bucketCount).toBe(0);
  });

  it('should handle non-zero start time correctly', () => {
    // 1000ms ticks. Cache starts at 1000ms (min log time is 500 -> 0 aligned... wait. 500/1000 floor is 0.)
    // Wait, the user manual test case had manual range. Now it's auto.
    // Logs: 500, 1500, 2500, 3500.
    // Min: 500. Aligned Min: floor(500/1000)*1000 = 0.
    // Max: 3500. Aligned Max: ceil(3500/1000)*1000 = 4000.
    const cache = createCacheWithLogs(
      [
        { severity: Severity.SeverityError, timeMs: 500 },
        { severity: Severity.SeverityError, timeMs: 1500 },
        { severity: Severity.SeverityInfo, timeMs: 2500 },
        { severity: Severity.SeverityError, timeMs: 3500 },
      ],
      1000,
    );

    // Query range: 2000ms - 4000ms (Window 1 and Window 2 relative to 0?)
    // alignedMinTimeMS = 0.
    // Request 2000-4000.
    // Window indices: (2000-0)/1000 = 2. (4000-0)/1000 = 4.
    // Indices 2, 3.
    // Log(2500) -> index 2.
    // Log(3500) -> index 3.
    // Log(1500) -> index 1.
    // Total logs in [2000, 4000): 2500 (Info), 3500 (Error). Total 2.

    const result = cache.getHistogramData(2000, 4000, 1000);

    // Index 0 (2000-3000): Info (1). Ratio 1/2.
    expect(result.logRatios[Severity.SeverityInfo][0]).toBeCloseTo(0.5);
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(0);

    // Index 1 (3000-4000): Error (1). Ratio 1/2.
    expect(result.logRatios[Severity.SeverityInfo][1]).toBeCloseTo(0);
    expect(result.logRatios[Severity.SeverityError][1]).toBeCloseTo(0.5);
  });

  it('should treat windowTimeMS smaller than minTickTimeMS as minTickTimeMS', () => {
    // 1000ms ticks.
    const cache = createCacheWithLogs(
      [{ severity: Severity.SeverityError, timeMs: 500 }],
      1000,
    );

    // Request window 500ms (< 1000ms)
    // Should behave as 1000ms window.
    const result = cache.getHistogramData(0, 1000, 500);

    // Window 0 (0-1000): 1 log. Total 1. Ratio 1.0.
    expect(result.logRatios[Severity.SeverityError][0]).toBeCloseTo(1.0);
    expect(result.bucketCount).toBe(1);
  });
});
