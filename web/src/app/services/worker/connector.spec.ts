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

import { delay, of } from 'rxjs';
import { TestingWorkerConnector, WorkerConnectorUtil } from './connector';
import { KHIWorkerPacket } from 'src/app/worker/worker-types';

interface TestingWorkerResponsePacket extends KHIWorkerPacket {
  foo: 'bar';
}

interface TestingWorkerLatencyRequestPacket extends KHIWorkerPacket {
  latency: number;
}

describe('WorkerConnectorUtil', () => {
  it('calls the worker connector and emits a result', (done) => {
    const connector = new TestingWorkerConnector((req) =>
      of({
        foo: 'bar',
        ...req,
      }),
    );
    WorkerConnectorUtil.unary<TestingWorkerResponsePacket>(
      connector,
      {},
    ).subscribe((res) => {
      expect(res.foo).toEqual('bar');
      done();
    });
  });

  it('completes after emitting a result', (done) => {
    const connector = new TestingWorkerConnector((req) =>
      of({
        foo: 'bar',
        ...req,
      }),
    );
    WorkerConnectorUtil.unary<TestingWorkerResponsePacket>(
      connector,
      {},
    ).subscribe({
      complete: () => {
        done();
      },
    });
  });

  it('emits the value on the observable returned from unary only for the observable associated with a request', (done) => {
    const connector = new TestingWorkerConnector((req) =>
      of({
        ...req,
      }).pipe(delay((req as TestingWorkerLatencyRequestPacket).latency)),
    );
    let callCount = 0;
    WorkerConnectorUtil.unary<TestingWorkerResponsePacket>(connector, {
      latency: 100,
    } as TestingWorkerLatencyRequestPacket).subscribe(() => {
      expect(callCount).toEqual(1);
      callCount++;
      done();
    });
    WorkerConnectorUtil.unary<TestingWorkerResponsePacket>(connector, {
      latency: 50,
    } as TestingWorkerLatencyRequestPacket).subscribe(() => {
      expect(callCount).toEqual(0);
      callCount++;
    });
  });
});
