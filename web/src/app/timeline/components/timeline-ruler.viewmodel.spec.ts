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

import { RulerViewModelBuilder } from './timeline-ruler.viewmodel';
import { HistogramCache, HistogramInfo } from './misc/histogram-cache';
import { Severity } from 'src/app/generated';

describe('RulerViewModelBuilder', () => {
  let builder: RulerViewModelBuilder;
  let mockAllLogsHistogramCache: jasmine.SpyObj<HistogramCache>;
  let mockFilteredLogsHistogramCache: jasmine.SpyObj<HistogramCache>;

  beforeEach(() => {
    builder = new RulerViewModelBuilder(300, 10);
    mockAllLogsHistogramCache = jasmine.createSpyObj('HistogramCache', [
      'getHistogramData',
    ]);
    mockFilteredLogsHistogramCache = jasmine.createSpyObj('HistogramCache', [
      'getHistogramData',
    ]);

    const severities = Object.values(Severity).filter(
      (s) => !isNaN(Number(s)),
    ) as Severity[];

    // Setup default mock returns
    const logRatios: { [key in Severity]: Float32Array } = {} as {
      [key in Severity]: Float32Array;
    };
    for (const s of severities) {
      logRatios[s] = new Float32Array(10).fill(0);
    }

    const defaultHistogramData: HistogramInfo = {
      bucketCount: 10,
      bucketTimeMs: 100,
      histogramBeginTimeMs: 0,
      maxBucketSumRatio: 1,
      totalLogCount: 0,
      logRatios: logRatios,
    };

    mockAllLogsHistogramCache.getHistogramData.and.returnValue(
      defaultHistogramData,
    );
    mockFilteredLogsHistogramCache.getHistogramData.and.returnValue(
      defaultHistogramData,
    );
  });

  describe('generateRulerViewModel', () => {
    it('populates histogram buckets from cache', () => {
      const bucketCount = 5;
      const logRatios: { [key in Severity]: Float32Array } = {} as {
        [key in Severity]: Float32Array;
      };
      const severities = Object.values(Severity).filter(
        (s) => !isNaN(Number(s)),
      ) as Severity[];
      for (const s of severities) {
        logRatios[s] = new Float32Array(10).fill(0);
      }
      logRatios[Severity.SeverityInfo][0] = 10;
      logRatios[Severity.SeverityWarning][1] = 5;

      const mockData = {
        bucketCount: bucketCount,
        bucketTimeMs: 100,
        histogramBeginTimeMs: 0,
        maxBucketSumRatio: 10,
        totalLogCount: 15,
        logRatios: logRatios,
      };

      mockAllLogsHistogramCache.getHistogramData.and.returnValue(mockData);
      mockFilteredLogsHistogramCache.getHistogramData.and.returnValue(mockData);

      const viewModel = builder.generateRulerViewModel(
        0,
        1,
        100,
        0,
        mockAllLogsHistogramCache,
        mockFilteredLogsHistogramCache,
      );

      expect(viewModel.histogramBuckets.length).toBe(bucketCount);
      // Check maxBucketSumRatio handling (normalization)
      // all.Info[0] should be 10 / 10 = 1.
      expect(viewModel.histogramBuckets[0].all[Severity.SeverityInfo]).toBe(1);
      // all.Warning[1] should be 5 / 10 = 0.5.
      expect(viewModel.histogramBuckets[1].all[Severity.SeverityWarning]).toBe(
        0.5,
      );
    });
  });
});
