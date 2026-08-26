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

import { computed, InjectionToken, Signal, signal } from '@angular/core';

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
   * Signal holding all current parameter values.
   */
  readonly currentParameters: Signal<{ [id: string]: unknown }>;

  /**
   * Signal holding the parameters verified by the last completed dryrun.
   */
  readonly validatedParameters: Signal<{ [id: string]: unknown }>;

  /**
   * Signal holding the default parameters returned by the backend.
   */
  readonly defaultParameters: Signal<{ [id: string]: unknown }>;

  /**
   * Returns a computed signal of the value for the given parameter ID.
   */
  get<T>(id: string): Signal<T>;

  /**
   * Returns a computed signal indicating if the field has pending unverified changes.
   */
  isValidating(id: string): Signal<boolean>;

  /**
   * Returns a computed signal indicating if the field was modified by the user.
   */
  isDirty(id: string): Signal<boolean>;

  /**
   * Set the value for the parameter with the given id.
   */
  set(id: string, value: unknown): void;

  /**
   * Set the default value of parameters.
   */
  setDefaultValues(defaultValues: { [id: string]: unknown }): void;

  /**
   * Set the snapshot of parameters that were validated by a completed dryrun.
   */
  setValidatedParameters(params: { [id: string]: unknown }): void;
}

/**
 * Checks if two parameter values are deeply equal (supporting primitives, arrays, and Dates).
 */
export function areValuesEqual(a: unknown, b: unknown): boolean {
  if (a === b) {
    return true;
  }
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) {
      return false;
    }
    for (let i = 0; i < a.length; i++) {
      if (a[i] !== b[i]) {
        return false;
      }
    }
    return true;
  }
  if (a instanceof Date && b instanceof Date) {
    return a.getTime() === b.getTime();
  }
  return false;
}

/**
 * Checks if two parameter objects have equal keys and values.
 */
export function haveEqualKeyValues(
  prev: { [id: string]: unknown },
  current: { [id: string]: unknown },
): boolean {
  const prevKeys = Object.keys(prev);
  const currentKeys = Object.keys(current);
  if (prevKeys.length !== currentKeys.length) {
    return false;
  }
  for (const key of prevKeys) {
    if (!(key in current) || !areValuesEqual(prev[key], current[key])) {
      return false;
    }
  }
  return true;
}

/**
 * Default implementation of ParameterStore backed by Angular Signals.
 */
export class DefaultParameterStore implements ParameterStore {
  readonly currentParameters = signal<{ [id: string]: unknown }>({});

  readonly validatedParameters = signal<{ [id: string]: unknown }>({});

  readonly defaultParameters = signal<{ [id: string]: unknown }>({});

  private readonly getSignals = new Map<string, Signal<unknown>>();

  private readonly validatingSignals = new Map<string, Signal<boolean>>();

  private readonly dirtySignals = new Map<string, Signal<boolean>>();

  private readonly dirtyFields = computed(() => {
    const current = this.currentParameters();
    const defaults = this.defaultParameters();
    const dirty = new Set<string>();
    for (const key of Object.keys(current)) {
      if (!areValuesEqual(current[key], defaults[key])) {
        dirty.add(key);
      }
    }
    return dirty;
  });

  /**
   * Returns a computed signal of the value for the given parameter ID.
   */
  get<T>(id: string): Signal<T> {
    let sig = this.getSignals.get(id);
    if (!sig) {
      sig = computed(() => this.currentParameters()[id]);
      this.getSignals.set(id, sig);
    }
    return sig as Signal<T>;
  }

  /**
   * Returns a computed signal indicating if the field's current value differs from the validated value.
   */
  isValidating(id: string): Signal<boolean> {
    let sig = this.validatingSignals.get(id);
    if (!sig) {
      sig = computed(() => {
        const current = this.currentParameters();
        const validated = this.validatedParameters();
        if (!(id in current)) {
          return false;
        }
        return !areValuesEqual(current[id], validated[id]);
      });
      this.validatingSignals.set(id, sig);
    }
    return sig;
  }

  /**
   * Returns a computed signal indicating if the field was modified by the user from its default value.
   */
  isDirty(id: string): Signal<boolean> {
    let sig = this.dirtySignals.get(id);
    if (!sig) {
      sig = computed(() => this.dirtyFields().has(id));
      this.dirtySignals.set(id, sig);
    }
    return sig;
  }

  /**
   * Sets the value for the parameter with the given ID.
   */
  set(id: string, value: unknown): void {
    this.currentParameters.update((prev) => ({
      ...prev,
      [id]: value,
    }));
  }

  /**
   * Updates default values for parameters.
   * Preserves user-modified (dirty) values while updating untouched fields.
   */
  setDefaultValues(defaultValues: { [id: string]: unknown }): void {
    const current = this.currentParameters();
    const dirty = this.dirtyFields();
    const next: { [id: string]: unknown } = {};
    for (const id of Object.keys({ ...current, ...defaultValues })) {
      if (dirty.has(id)) {
        next[id] = current[id];
      } else {
        next[id] = defaultValues[id];
      }
    }
    this.currentParameters.set(next);
    this.defaultParameters.set(defaultValues);
  }

  /**
   * Sets the parameters that have been validated by a completed dryrun.
   */
  setValidatedParameters(params: { [id: string]: unknown }): void {
    this.validatedParameters.set({ ...params });
  }

  /**
   * Unregisters resources when the store is destroyed.
   */
  public destroy(): void {
    this.getSignals.clear();
    this.validatingSignals.clear();
    this.dirtySignals.clear();
  }
}
