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

import { LogStore, LogDTO } from 'src/app/store/domain/log-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';

describe('LogStore', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let store: LogStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = InternPoolStore.create();
    styleStore = new StyleStore();
    store = LogStore.create(internPool, styleStore, 0);

    // Avoid errors of missing keys in basic tests
    styleStore.addSeverities([
      {
        id: 1,
        label: 'S1',
        shortLabel: 'S1',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
      {
        id: 2,
        label: 'S2',
        shortLabel: 'S2',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
      {
        id: 3,
        label: 'S3',
        shortLabel: 'S3',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
      {
        id: 4,
        label: 'S4',
        shortLabel: 'S4',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
    ]);

    styleStore.addLogTypes([
      {
        id: 1,
        label: 'L1',
        description: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
      {
        id: 2,
        label: 'L2',
        description: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
      {
        id: 3,
        label: 'L3',
        description: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
      {
        id: 4,
        label: 'L4',
        description: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
    ]);
  });

  it('should succeed with correctly ordered timestamps', () => {
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 1005n, logTypeId: 2, severityTypeId: 2, summaryStringId: 2 },
      { id: 3, ts: 1005n, logTypeId: 3, severityTypeId: 3, summaryStringId: 3 },
      { id: 4, ts: 1010n, logTypeId: 4, severityTypeId: 4, summaryStringId: 4 },
    ];

    expect(() => {
      const s = LogStore.create(internPool, styleStore, 4);
      s.addLogs(logs);
      s.shrinkToFit();
    }).not.toThrow();
  });

  it('should throw error if logs are out of timestamp order', () => {
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 999n, logTypeId: 2, severityTypeId: 2, summaryStringId: 2 },
    ];

    expect(() => {
      const s = LogStore.create(internPool, styleStore, 2);
      s.addLogs(logs);
    }).toThrowError(/Logs are not sorted by timestamp/);
  });

  it('should fetch log entries and handle incorrect id lookups', () => {
    internPool.addStrings([
      { id: 1, value: 'first_summary' },
      { id: 2, value: 'second_summary' },
    ]);

    styleStore.addSeverities([
      {
        id: 10,
        label: 'INFO',
        shortLabel: 'I',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
    ]);

    styleStore.addLogTypes([
      {
        id: 100,
        label: 'audit',
        description: 'audit desc',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
    ]);

    const logs: LogDTO[] = [
      {
        id: 55,
        ts: 10n,
        logTypeId: 100,
        severityTypeId: 10,
        summaryStringId: 1,
        bodyStructId: 123,
      },
      {
        id: 56,
        ts: 20n,
        logTypeId: 100,
        severityTypeId: 10,
        summaryStringId: 2,
      },
    ];

    store = LogStore.create(internPool, styleStore, 2);
    store.addLogs(logs);
    store.shrinkToFit();

    const logObj = store.getLog(55);
    expect(logObj.id).toBe(55);
    expect(logObj.timestamp).toBe(10n);
    expect(logObj.summary).toBe('first_summary');
    expect(logObj.severity.label).toBe('INFO');
    expect(logObj.logType.label).toBe('audit');
    expect(logObj.logIndex).toBe(0);
    expect(logObj.structId).toBe(123);

    expect(store.getIndex(55)).toBe(0);
    expect(store.getIndex(56)).toBe(1);
    expect(() => store.getIndex(99)).toThrowError('Log ID 99 not found');

    expect(() => store.getLog(99)).toThrowError('Log ID 99 not found');
  });

  it('should return count and iterator correctly', () => {
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 1005n, logTypeId: 2, severityTypeId: 2, summaryStringId: 2 },
    ];

    store = LogStore.create(internPool, styleStore, 2);
    store.addLogs(logs);
    store.shrinkToFit();

    expect(store.count).toBe(2);

    const iteratedLogs = Array.from(store.logs());
    expect(iteratedLogs.length).toBe(2);
    expect(iteratedLogs[0].id).toBe(1);
    expect(iteratedLogs[1].id).toBe(2);
  });

  describe('create and dynamic methods', () => {
    it('should create an empty store and add logs dynamically', () => {
      const dynamicStore = LogStore.create(internPool, styleStore, 1);
      expect(dynamicStore.count).toBe(0);

      dynamicStore.addLog({
        id: 10,
        ts: 100n,
        logTypeId: 100,
        severityTypeId: 10,
        summaryStringId: 1,
      });
      dynamicStore.addLog({
        id: 20,
        ts: 200n,
        logTypeId: 100,
        severityTypeId: 10,
        summaryStringId: 2,
        bodyStructId: 42,
      });

      expect(dynamicStore.count).toBe(2);
      expect(dynamicStore.getLog(10).timestamp).toBe(100n);
      expect(dynamicStore.getLog(20).timestamp).toBe(200n);
      expect(dynamicStore.getLog(20).structId).toBe(42);

      dynamicStore.shrinkToFit();
      expect(dynamicStore.count).toBe(2);
      expect(dynamicStore.getLog(10).timestamp).toBe(100n);
      expect(dynamicStore.getLog(20).timestamp).toBe(200n);
    });

    it('should handle multiple buffer expansions starting from capacity 1 and preserve all fields', () => {
      for (let i = 1; i <= 20; i++) {
        internPool.addString({ id: i, value: `summary_${i}` });
      }

      const dynamicStore = LogStore.create(internPool, styleStore, 1);
      for (let i = 1; i <= 20; i++) {
        dynamicStore.addLog({
          id: i,
          ts: BigInt(i * 10),
          logTypeId: ((i - 1) % 4) + 1,
          severityTypeId: ((i - 1) % 4) + 1,
          summaryStringId: i,
          bodyStructId: i * 100,
        });
      }

      expect(dynamicStore.count).toBe(20);
      for (let i = 1; i <= 20; i++) {
        const log = dynamicStore.getLog(i);
        expect(log.id).toBe(i);
        expect(log.timestamp).toBe(BigInt(i * 10));
        expect(log.summary).toBe(`summary_${i}`);
        expect(log.logType.label).toBe(`L${((i - 1) % 4) + 1}`);
        expect(log.severity.label).toBe(`S${((i - 1) % 4) + 1}`);
        expect(log.structId).toBe(i * 100);
      }

      dynamicStore.shrinkToFit();
      // Should be idempotent on multiple calls
      dynamicStore.shrinkToFit();
      expect(dynamicStore.count).toBe(20);
      for (let i = 1; i <= 20; i++) {
        const log = dynamicStore.getLog(i);
        expect(log.id).toBe(i);
        expect(log.timestamp).toBe(BigInt(i * 10));
        expect(log.summary).toBe(`summary_${i}`);
        expect(log.structId).toBe(i * 100);
      }
    });

    it('should handle shrinkToFit on an empty store with capacity 0', () => {
      const emptyStore = LogStore.create(internPool, styleStore, 0);
      emptyStore.shrinkToFit();
      expect(emptyStore.count).toBe(0);
      expect(() => emptyStore.getLog(1)).toThrowError(/Log ID 1 not found/);
    });

    it('should throw error when adding unsorted logs', () => {
      const dynamicStore = LogStore.create(internPool, styleStore);
      dynamicStore.addLog({
        id: 1,
        ts: 200n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      });

      expect(() =>
        dynamicStore.addLog({
          id: 2,
          ts: 100n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 1,
        }),
      ).toThrowError(/Logs are not sorted by timestamp/);
    });

    it('should retrieve log ID by chronological index', () => {
      const testStore = LogStore.create(internPool, styleStore);
      // Log IDs (5, 2, 10) are not in chronological order, but timestamps (100n, 200n, 300n) are
      testStore.addLog({
        id: 5,
        ts: 100n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      });
      testStore.addLog({
        id: 2,
        ts: 200n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      });
      testStore.addLog({
        id: 10,
        ts: 300n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      });

      expect(testStore.getLogIdByIndex(0)).toBe(5);
      expect(testStore.getLogIdByIndex(1)).toBe(2);
      expect(testStore.getLogIdByIndex(2)).toBe(10);
      expect(() => testStore.getLogIdByIndex(3)).toThrowError(
        /Log index 3 out of bounds/,
      );
      expect(() => testStore.getLogIdByIndex(-1)).toThrowError(
        /Log index -1 out of bounds/,
      );
    });
  });
});
