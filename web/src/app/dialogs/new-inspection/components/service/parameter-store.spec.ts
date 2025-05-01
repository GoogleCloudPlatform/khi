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

import { firstValueFrom, lastValueFrom, take } from 'rxjs';
import { DefaultParameterStore } from './parameter-store';

describe('DefaultParameterStore', () => {
  let parameterStore: DefaultParameterStore;

  beforeEach(() => {
    parameterStore = new DefaultParameterStore();
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  describe('watch', () => {
    it('emits value set before the subscribe', async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');

      expect(await firstValueFrom(parameterStore.watch('foo'))).toBe('qux');
    });
  });

  describe('set', () => {
    it('put the value after the default value being available', async () => {
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(
        await lastValueFrom(parameterStore.watch('foo').pipe(take(1))),
      ).toBe('qux');
    });

    it('put the value after the default value being available and keep the order', async () => {
      parameterStore.set('foo', 'qux');
      parameterStore.set('foo', 'quux');
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(
        await lastValueFrom(parameterStore.watch('foo').pipe(take(1))),
      ).toBe('quux');
    });
  });

  describe('setDefaultValues', () => {
    it("doesn't overwrite the value set before", async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'bar', bar: 'qux' });

      expect(
        await lastValueFrom(parameterStore.watch('foo').pipe(take(1))),
      ).toBe('qux');
    });

    it('update the value if the previous value is same as the default value', async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'bar');
      parameterStore.setDefaultValues({ foo: 'qux' });

      expect(
        await lastValueFrom(parameterStore.watch('foo').pipe(take(1))),
      ).toBe('qux');
    });
  });

  describe('watchDirty', () => {
    it("becomes false when the value hasn't set", async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(
        await lastValueFrom(parameterStore.watchDirty('foo').pipe(take(1))),
      ).toBe(false);
    });

    it('becomes false when the value has set', async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');

      expect(
        await lastValueFrom(parameterStore.watchDirty('foo').pipe(take(1))),
      ).toBe(true);
    });

    it('becomes false when the previous value returned to the same as the default value', async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.set('foo', 'bar');

      expect(
        await lastValueFrom(parameterStore.watchDirty('foo').pipe(take(1))),
      ).toBe(false);
    });

    it('becomes false when the default value was updated and it matched the current value', async () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'qux' });

      expect(
        await lastValueFrom(parameterStore.watchDirty('foo').pipe(take(1))),
      ).toBe(false);
    });
  });
});
