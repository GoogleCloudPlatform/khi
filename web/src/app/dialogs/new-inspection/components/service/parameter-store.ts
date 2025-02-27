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

import { InjectionToken } from '@angular/core';
import {
  distinctUntilChanged,
  filter,
  map,
  Observable,
  ReplaySubject,
  Subject,
  take,
  takeUntil,
} from 'rxjs';

/**
 * The injection token to get an implementation of ParameterStore.
 */
export const PARAMETER_STORE = new InjectionToken<ParameterStore>(
  'PARAMETER_STORE',
);

/**
 * ParameterStore is an interface to store the parameter values of the new-inspection dialog.
 */
export interface ParameterStore {
  /**
   * Get the observable to monitor the value of speicifed value.
   * When the specified id is not available parameter for now, it waits the value to be available.
   */
  watch<T>(id: string): Observable<T>;

  /**
   * Watch the all parameters.
   */
  watchAll(): Observable<{ [id: string]: unknown }>;

  /**
   * Set the value for the parameter with the given id.
   */
  set(id: string, value: unknown): void;

  /**
   * Set the default value of parameters.
   */
  setDefaultValues(defaultValues: { [id: string]: unknown }): void;
}

export class DefaultParameterStore implements ParameterStore {
  readonly destroyed = new Subject();

  readonly currentParameters = new ReplaySubject<{ [id: string]: unknown }>(1);

  constructor() {
    this.currentParameters.next({});
  }
  watchAll(): Observable<{ [id: string]: unknown }> {
    return this.currentParameters.pipe(
      takeUntil(this.destroyed),
      distinctUntilChanged((prev, current) => {
        return this.haveEqualKeyValues(prev, current);
      }),
    );
  }

  watch<T>(id: string): Observable<T> {
    return this.currentParameters.pipe(
      takeUntil(this.destroyed),
      filter((parameters) => id in parameters),
      map((parameters) => parameters[id] as T),
      distinctUntilChanged(),
    );
  }

  set(id: string, value: unknown): void {
    this.currentParameters
      .pipe(
        takeUntil(this.destroyed),
        filter((parameters) => id in parameters),
        take(1),
      )
      .subscribe((parameters) => {
        this.currentParameters.next({
          ...parameters,
          [id]: value,
        });
      });
  }

  setDefaultValues(defaultValues: { [id: string]: unknown }): void {
    this.currentParameters
      .pipe(takeUntil(this.destroyed), take(1))
      .subscribe((parameters) => {
        this.currentParameters.next({ ...defaultValues, ...parameters });
      });
  }

  /**
   * Unregister subscriptions registered in this store.
   */
  public destroy(): void {
    this.destroyed.next(void 0);
  }

  /**
   * Check if the given objects are both having same key and its value.
   * This doesn't compare them recursively, because currently the parameter values are all primitives.
   */
  private haveEqualKeyValues(
    prev: { [id: string]: unknown },
    current: { [id: string]: unknown },
  ): boolean {
    for (const prevFieldKey in prev) {
      if (
        !(prevFieldKey in current) ||
        prev[prevFieldKey] !== current[prevFieldKey]
      ) {
        return false;
      }
    }
    for (const currentFieldKey in current) {
      if (!(currentFieldKey in prev)) {
        return false;
      }
    }
    return true;
  }
}
