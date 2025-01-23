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

import { delay, filter, first, map, Observable, Subject, tap } from 'rxjs';
import { randomString } from 'src/app/utils/random';
import {
  isKHIWorkerPacket,
  KHIWorkerPacket,
} from 'src/app/worker/worker-types';

/**
 * An interface provides the feature to communicate with the other worker possibly handled by WebWorker or backend.
 */
export interface WorkerConnector {
  /**
   * Requst this worker with input.
   */
  request(req: object): void;

  /**
   * Get the observable emits MessageEvent from the worker.
   */
  messages(): Observable<MessageEvent>;
}

/**
 * An implementaion of WorkerConnector using WebWorker script.
 */
export class WebWorkerConnector implements WorkerConnector {
  private readonly worker: Worker;

  private readonly messageSubject: Subject<MessageEvent> = new Subject();

  constructor(worker: Worker) {
    this.worker = worker;
    this.worker.onmessage = (msg) => {
      if (isKHIWorkerPacket(msg.data)) {
        this.messageSubject.next(msg);
      }
    };
  }

  request(req: object): void {
    this.worker.postMessage(req);
  }
  messages(): Observable<MessageEvent> {
    return this.messageSubject;
  }
}

/**
 * An implementation of WorkerConnector for testing purpose.
 */
export class TestingWorkerConnector implements WorkerConnector {
  private readonly messageSubject: Subject<MessageEvent> = new Subject();

  constructor(
    private readonly mockWorker: (req: object) => Observable<object>,
  ) {}
  request(req: object): void {
    // request is postponed until the next frame to emulate the behavior with the real web worker.
    this.mockWorker(req)
      .pipe(first(), delay(0)) // request must not be routed to the messages observable immediately.
      .subscribe((data) => {
        this.messageSubject.next({
          data: data,
        } as MessageEvent);
      });
  }
  messages(): Observable<MessageEvent> {
    return this.messageSubject;
  }
}

export class WorkerConnectorUtil {
  /**
   * Calls a worker API accepts a request and returns a reply.
   */
  public static unary<Response>(
    connector: WorkerConnector,
    req: KHIWorkerPacket,
  ): Observable<Response> {
    const taskId = randomString();
    req.taskId = taskId;
    req.isKHIWorkerPacket = true;
    return connector.messages().pipe(
      filter((msg) => msg.data.taskId === taskId),
      map((msg) => msg.data as Response),
      first(),
      tap({
        subscribe: () => {
          connector.request(req);
        },
      }),
    );
  }
}
