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

import { BehaviorSubject, ReplaySubject, Subject, filter } from 'rxjs';

/**
 * An abstract data storage got from inter-frame connection.
 */
export abstract class InterframeDatasource<T> {
  /**
   * Determine wheather the data update request from main frame should be accepted or not.
   */
  public readonly bound$ = new BehaviorSubject(true);

  /**
   * Must be updated by the child class.
   * The updated data sent from main frame should be routed to here.
   */
  protected readonly rawUpdateRequest$ = new Subject<T>();

  /**
   * An observable to monitor the data change.
   * Data update won't be reported when bound$ is false.
   */
  public readonly data$ = new ReplaySubject<T>(1);

  constructor() {
    this.rawUpdateRequest$
      .pipe(filter(() => this.bound$.value))
      .subscribe((data) => {
        this.data$.next(data);
      });
  }

  abstract enable(): void;

  abstract disable(): void;
}
