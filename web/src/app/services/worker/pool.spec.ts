/**
 * Copyright 2025 Google LLC
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

import { delay, firstValueFrom, Observable, of, reduce } from 'rxjs';
import { TestingWorkerConnector } from './connector';
import { KHIWorkerPacket } from 'src/app/worker/worker-types';
import { ConnectorPool } from './pool';

describe('ConnectorPool', () => {
  it('processes requests in series using available workers', async () => {
    const mockWorker1 = new TestingWorkerConnector((req: object) => {
      return of({
        ...req,
        worker: '1',
      }).pipe(delay(Math.random() * 100));
    });
    const mockWorker2 = new TestingWorkerConnector((req: object) => {
      return of({
        ...req,
        worker: '2',
      }).pipe(delay(Math.random() * 100));
    });

    const connectorPool = new ConnectorPool([mockWorker1, mockWorker2]);

    const requests = [
      { value: 'request1' },
      { value: 'request2' },
      { value: 'request3' },
      { value: 'request4' },
    ];

    const requestFinalizer = (request: {
      value: string;
    }): Observable<KHIWorkerPacket> => {
      return of({
        request: request.value,
        response: `response of ${request.value}`,
      } as KHIWorkerPacket).pipe(delay(Math.random() * 100));
    };

    const resultsPromise = firstValueFrom(
      connectorPool.requestSeriesOfTasks(requests, requestFinalizer).pipe(
        // collect all results into a single array
        reduce(
          (acc: ({ request: string; response: string } | null)[], curr) => {
            acc[curr.index] = curr.response as {
              request: string;
              response: string;
            };
            return acc;
          },
          [null, null, null, null] as ({
            request: string;
            response: string;
          } | null)[],
        ),
      ),
    );

    const results = await resultsPromise;

    expect(results.length).toBe(4);
    expect(results[0]!.request).toBe('request1');
    expect(results[1]!.request).toBe('request2');
    expect(results[2]!.request).toBe('request3');
    expect(results[3]!.request).toBe('request4');
    expect(results[0]!.response).toBe('response of request1');
    expect(results[1]!.response).toBe('response of request2');
    expect(results[2]!.response).toBe('response of request3');
    expect(results[3]!.response).toBe('response of request4');
  });

  it('handles empty request array', async () => {
    const connectorPool = new ConnectorPool([]);
    const requests: unknown[] = [];
    const requestFinalizer = () => of({} as KHIWorkerPacket);
    const results = await firstValueFrom(
      connectorPool.requestSeriesOfTasks(requests, requestFinalizer).pipe(
        reduce((acc, curr) => {
          acc.push(curr);
          return acc;
        }, [] as unknown[]),
      ),
    );
    expect(results).toEqual([]);
  });
});
