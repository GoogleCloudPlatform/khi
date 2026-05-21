/**
 * Copyright 2026 Google LLC
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

import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';

describe('InternPoolStore', () => {
  let store: InternPoolStore;

  beforeEach(() => {
    store = new InternPoolStore();
  });

  it('should add and get strings from the pool', () => {
    store.addStrings([
      { id: 1, value: 'foo' },
      { id: 3, value: 'bar' },
    ]);

    expect(store.getString(1)).toBe('foo');
    expect(store.getString(3)).toBe('bar');
  });

  it('should throw an error if string ID is missing', () => {
    expect(() => store.getString(999)).toThrowError(
      'String ID 999 not found in pool',
    );
  });

  it('should add and get field path sets', () => {
    store.addStrings([
      { id: 1, value: 'foo' },
      { id: 2, value: 'bar' },
      { id: 3, value: 'baz' },
      { id: 4, value: 'alpha' },
      { id: 5, value: 'beta' },
    ]);

    store.addFieldPathSets([
      { id: 10, fieldPathStringIds: [1, 2, 3] },
      { id: 20, fieldPathStringIds: [4, 5] },
    ]);

    expect(store.getFieldPathSet(10)).toEqual(['foo', 'bar', 'baz']);
    expect(store.getFieldPathSet(20)).toEqual(['alpha', 'beta']);
  });

  it('should throw an error if field path set ID is missing', () => {
    expect(() => store.getFieldPathSet(999)).toThrowError(
      'FieldPathSet ID 999 not found in pool',
    );
  });
});
