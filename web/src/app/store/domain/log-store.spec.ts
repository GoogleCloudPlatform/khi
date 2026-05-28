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
import { create, toBinary } from '@bufbuild/protobuf';
import {
  InternedStructSchema,
  InternedValueSchema,
} from 'src/app/generated/khifile/shared_pb';

describe('LogStore', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let store: LogStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = new InternPoolStore();
    styleStore = new StyleStore();
    store = new LogStore(internPool, styleStore);

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

    expect(() => store.initialize(logs, 4)).not.toThrow();
  });

  it('should throw error if logs are out of timestamp order', () => {
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 999n, logTypeId: 2, severityTypeId: 2, summaryStringId: 2 },
    ];

    expect(() => store.initialize(logs, 2)).toThrowError(
      /Logs are not sorted by timestamp/,
    );
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
      },
      {
        id: 56,
        ts: 20n,
        logTypeId: 100,
        severityTypeId: 10,
        summaryStringId: 2,
      },
    ];

    store.initialize(logs, 2);

    const logObj = store.getLog(55);
    expect(logObj.id).toBe(55);
    expect(logObj.timestamp).toBe(10n);
    expect(logObj.summary).toBe('first_summary');
    expect(logObj.severity.label).toBe('INFO');
    expect(logObj.logType.label).toBe('audit');
    expect(logObj.logIndex).toBe(0);

    expect(store.getIndex(55)).toBe(0);
    expect(store.getIndex(56)).toBe(1);
    expect(() => store.getIndex(99)).toThrowError('Log ID 99 not found');

    expect(() => store.getLog(99)).toThrowError('Log ID 99 not found');
  });

  it('should return decoded log body correctly', () => {
    internPool.addStrings([
      { id: 10, value: 'user' },
      { id: 11, value: 'status' },
      { id: 12, value: 'alice' },
    ]);

    internPool.addFieldPathSets([{ id: 1, fieldPathStringIds: [10, 11] }]);

    const struct = create(InternedStructSchema, {
      fieldPathSetId: 1,
      values: [
        create(InternedValueSchema, {
          kind: { case: 'stringValue', value: 12 },
        }),
        create(InternedValueSchema, {
          kind: { case: 'int64Value', value: 42n },
        }),
      ],
    });

    const logs: LogDTO[] = [
      {
        id: 1,
        ts: 10n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
        body: toBinary(InternedStructSchema, struct),
      },
      { id: 2, ts: 20n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
    ];

    store.initialize(logs, 2);

    expect(store.getLog(1).body).toEqual({
      user: 'alice',
      status: 42,
    });
    expect(store.getLog(1).bodyYAML).toBe('user: alice\nstatus: 42\n');
    expect(store.getLog(2).body).toBeNull();
    expect(store.getLog(2).bodyYAML).toBe('');
    expect(() => store.getLog(99)).toThrowError('Log ID 99 not found');
  });

  it('should return count and iterator correctly', () => {
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 1005n, logTypeId: 2, severityTypeId: 2, summaryStringId: 2 },
    ];

    store.initialize(logs, 2);

    expect(store.count).toBe(2);

    const iteratedLogs = Array.from(store.logs());
    expect(iteratedLogs.length).toBe(2);
    expect(iteratedLogs[0].id).toBe(1);
    expect(iteratedLogs[1].id).toBe(2);
  });
});
