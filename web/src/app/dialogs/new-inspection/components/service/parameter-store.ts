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
  withLatestFrom,
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
   * Get the observable to monitor the dirtiness of the field.
   * true is emitted after user modified the field.
   */
  watchDirty(id: string): Observable<boolean>;

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

  readonly currentDefaultParameters = new ReplaySubject<{
    [id: string]: unknown;
  }>(1);

  private dirtyFields = this.currentParameters.pipe(
    takeUntil(this.destroyed),
    withLatestFrom(this.currentDefaultParameters),
    map(
      ([parameters, defaultParameters]) =>
        new Set(
          Object.keys(parameters).filter(
            (key) => parameters[key] !== defaultParameters[key],
          ),
        ),
    ),
  );
  constructor() {
    this.currentParameters.next({});
    this.currentDefaultParameters.next({});
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

  watchDirty(id: string): Observable<boolean> {
    return this.dirtyFields.pipe(
      takeUntil(this.destroyed),
      map((dirtyFields) => dirtyFields.has(id)),
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

  /**
   * Update the default values for parameters.
   * If the current value is same as the previous default values, the parameter is updated with the newer default values.
   * If not, the parameter value is kept because it was updated by the user.
   */
  setDefaultValues(defaultValues: { [id: string]: unknown }): void {
    this.currentParameters
      .pipe(
        takeUntil(this.destroyed),
        take(1),
        withLatestFrom(this.dirtyFields),
      )
      .subscribe(([parameters, dirtyFields]) => {
        const nextParameter: { [id: string]: unknown } = {};
        for (const id of Object.keys({ ...parameters, ...defaultValues })) {
          if (dirtyFields.has(id)) {
            nextParameter[id] = parameters[id];
          } else {
            nextParameter[id] = defaultValues[id];
          }
        }
        this.currentParameters.next(nextParameter);
      });
    this.currentDefaultParameters.next(defaultValues);
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
