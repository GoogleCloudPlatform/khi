/**
 * Copyright 2024 Google LLC
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

import { of } from 'rxjs';
import { ReferenceResolverStore } from '../common/loader/reference-resolver';
import { LogEntry } from '../store/log';
import { FilterWorkerServieUtil } from './filter-worker.service';

describe('FilterWorkerServiceUtil', () => {
  describe('logEntriesToFilterWorkerLogs', () => {
    it('should convert LogEntry to FilterWorkerLog', (done) => {
      const resolver = jasmine.createSpyObj<ReferenceResolverStore>(
        'ReferenceResolverStore',
        ['getText'],
      );
      const logEntry1 = {
        logIndex: 1,
        body: { type: 0 },
        summary: 'bar',
      } as LogEntry;
      const logEntry2 = {
        logIndex: 2,
        body: { type: 0 },
        summary: 'bar',
      } as LogEntry;
      resolver.getText.and.returnValue(of('foo'));

      FilterWorkerServieUtil.logEntriesToFilterWorkerLogs(resolver, [
        logEntry1,
        logEntry2,
      ]).subscribe((result) => {
        expect(result).toEqual([
          { index: 1, logBody: 'foo', logSummary: 'bar' },
          { index: 2, logBody: 'foo', logSummary: 'bar' },
        ]);
        done();
      });
    });
  });
});
