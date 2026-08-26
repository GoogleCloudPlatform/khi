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

import {
  areValuesEqual,
  DefaultParameterStore,
  haveEqualKeyValues,
} from './parameter-store';

describe('DefaultParameterStore', () => {
  let parameterStore: DefaultParameterStore;

  beforeEach(() => {
    parameterStore = new DefaultParameterStore();
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  describe('get', () => {
    it('returns the value set in the store', () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');

      expect(parameterStore.get('foo')()).toBe('qux');
    });

    it('returns the same signal instance on subsequent calls for the same id', () => {
      const sig1 = parameterStore.get('foo');
      const sig2 = parameterStore.get('foo');

      expect(sig1).toBe(sig2);
    });
  });

  describe('set', () => {
    it('puts the value after the default value is available', () => {
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(parameterStore.get('foo')()).toBe('qux');
    });

    it('puts the value after the default value is available and keeps the latest value', () => {
      parameterStore.set('foo', 'qux');
      parameterStore.set('foo', 'quux');
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(parameterStore.get('foo')()).toBe('quux');
    });
  });

  describe('setDefaultValues', () => {
    it("doesn't overwrite the value set before if user modified it", () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'bar', bar: 'qux' });

      expect(parameterStore.get('foo')()).toBe('qux');
    });

    it('updates the value if the previous value is same as the default value', () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'bar');
      parameterStore.setDefaultValues({ foo: 'qux' });

      expect(parameterStore.get('foo')()).toBe('qux');
    });
  });

  describe('isDirty', () => {
    it("is false when the value hasn't been modified from default", () => {
      parameterStore.setDefaultValues({ foo: 'bar' });

      expect(parameterStore.isDirty('foo')()).toBe(false);
    });

    it('is true when the value has been modified from default', () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');

      expect(parameterStore.isDirty('foo')()).toBe(true);
    });

    it('becomes false when the value returns to the same as default value', () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.set('foo', 'bar');

      expect(parameterStore.isDirty('foo')()).toBe(false);
    });

    it('becomes false when default value is updated and matches the current value', () => {
      parameterStore.setDefaultValues({ foo: 'bar' });
      parameterStore.set('foo', 'qux');
      parameterStore.setDefaultValues({ foo: 'qux' });

      expect(parameterStore.isDirty('foo')()).toBe(false);
    });
    it('handles array values by content without falsely marking as dirty', () => {
      parameterStore.setDefaultValues({ foo: ['a', 'b'] });

      expect(parameterStore.isDirty('foo')()).toBe(false);

      parameterStore.set('foo', ['a']);
      expect(parameterStore.isDirty('foo')()).toBe(true);

      parameterStore.set('foo', ['a', 'b']);
      expect(parameterStore.isDirty('foo')()).toBe(false);
    });

    it('returns the same signal instance on subsequent calls for the same id', () => {
      const sig1 = parameterStore.isDirty('foo');
      const sig2 = parameterStore.isDirty('foo');

      expect(sig1).toBe(sig2);
    });
  });

  describe('isValidating', () => {
    it('is false initially when field is not in store', () => {
      expect(parameterStore.isValidating('foo')()).toBe(false);
    });

    it('is true when current value differs from validated value', () => {
      parameterStore.set('foo', 'bar');
      parameterStore.setValidatedParameters({ foo: 'old' });

      expect(parameterStore.isValidating('foo')()).toBe(true);
    });

    it('becomes false when validated value matches current value', () => {
      parameterStore.set('foo', 'bar');
      parameterStore.setValidatedParameters({ foo: 'bar' });

      expect(parameterStore.isValidating('foo')()).toBe(false);
    });

    it('returns the same signal instance on subsequent calls for the same id', () => {
      const sig1 = parameterStore.isValidating('foo');
      const sig2 = parameterStore.isValidating('foo');

      expect(sig1).toBe(sig2);
    });

    it('compares array values by content, not reference', () => {
      parameterStore.set('foo', ['item1', 'item2']);
      parameterStore.setValidatedParameters({ foo: ['item1', 'item2'] });

      expect(parameterStore.isValidating('foo')()).toBe(false);

      parameterStore.set('foo', ['item1']);
      expect(parameterStore.isValidating('foo')()).toBe(true);
    });
  });

  describe('areValuesEqual and haveEqualKeyValues', () => {
    it('correctly compares array and primitive equality', () => {
      expect(areValuesEqual(['a', 'b'], ['a', 'b'])).toBe(true);
      expect(areValuesEqual(['a', 'b'], ['a', 'c'])).toBe(false);
      expect(areValuesEqual(['a'], ['a', 'b'])).toBe(false);
      expect(areValuesEqual('hello', 'hello')).toBe(true);
      expect(areValuesEqual('hello', 'world')).toBe(false);

      expect(haveEqualKeyValues({ a: ['x'] }, { a: ['x'] })).toBe(true);
      expect(haveEqualKeyValues({ a: ['x'] }, { a: ['y'] })).toBe(false);
    });
  });
});
