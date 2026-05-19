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
import {
  InternedStructDecoder,
  InternedStructDTO,
} from 'src/app/store/domain/struct-decoder';

describe('InternedStructDecoder', () => {
  let store: InternPoolStore;
  let decoder: InternedStructDecoder;

  beforeEach(() => {
    store = new InternPoolStore();
    decoder = new InternedStructDecoder(store);
  });

  it('should decode simple struct value variants', () => {
    store.addStrings([
      { id: 1, value: 'a' },
      { id: 2, value: 'b' },
      { id: 3, value: 'c' },
      { id: 4, value: 'd' },
      { id: 5, value: 'e' },
      { id: 6, value: 'f' },
      { id: 100, value: 'resolved_value' },
    ]);

    store.addFieldPathSets([
      { id: 10, fieldPathStringIds: [1, 2, 3, 4, 5, 6] },
    ]);

    const struct: InternedStructDTO = {
      fieldPathSetId: 10,
      values: [
        { kind: { case: 'nullValue', value: null } },
        { kind: { case: 'int64Value', value: 123 } },
        { kind: { case: 'doubleValue', value: 3.14 } },
        { kind: { case: 'boolValue', value: true } },
        { kind: { case: 'stringValue', value: 100 } },
        { kind: { case: 'timestampValue', value: 1000000000500n } },
      ],
    };

    const result = decoder.decode(struct);

    expect(result).toEqual({
      a: null,
      b: 123,
      c: 3.14,
      d: true,
      e: 'resolved_value',
      f: 1000000000500n,
    });
  });

  it('should decode flattened keys with \\0 separators correctly', () => {
    store.addStrings([
      { id: 1, value: 'metadata\0name' },
      { id: 2, value: 'metadata\0labels\0env' },
    ]);

    store.addFieldPathSets([{ id: 20, fieldPathStringIds: [1, 2] }]);

    store.addStrings([
      { id: 10, value: 'frontend-pod' },
      { id: 11, value: 'production' },
    ]);

    const struct: InternedStructDTO = {
      fieldPathSetId: 20,
      values: [
        { kind: { case: 'stringValue', value: 10 } },
        { kind: { case: 'stringValue', value: 11 } },
      ],
    };

    const result = decoder.decode(struct);

    expect(result).toEqual({
      metadata: {
        name: 'frontend-pod',
        labels: {
          env: 'production',
        },
      },
    });
  });

  it('should decode recursive nested structs and list value variants', () => {
    store.addStrings([
      { id: 1, value: 'nested' },
      { id: 2, value: 'elements' },
      { id: 3, value: 'key' },
      { id: 10, value: 'leaf' },
    ]);

    store.addFieldPathSets([
      { id: 30, fieldPathStringIds: [1, 2] },
      { id: 31, fieldPathStringIds: [3] },
    ]);

    const nestedStruct: InternedStructDTO = {
      fieldPathSetId: 31,
      values: [{ kind: { case: 'stringValue', value: 10 } }],
    };

    const struct: InternedStructDTO = {
      fieldPathSetId: 30,
      values: [
        { kind: { case: 'structValue', value: nestedStruct } },
        {
          kind: {
            case: 'listValue',
            value: {
              values: [
                { kind: { case: 'int64Value', value: 10 } },
                { kind: { case: 'boolValue', value: false } },
              ],
            },
          },
        },
      ],
    };

    const result = decoder.decode(struct);

    expect(result).toEqual({
      nested: {
        key: 'leaf',
      },
      elements: [10, false],
    });
  });

  it('should throw when length of field path does not match values length', () => {
    store.addStrings([
      { id: 1, value: 'key_one' },
      { id: 2, value: 'key_two' },
    ]);

    store.addFieldPathSets([{ id: 40, fieldPathStringIds: [1, 2] }]);

    const struct: InternedStructDTO = {
      fieldPathSetId: 40,
      values: [{ kind: { case: 'boolValue', value: true } }],
    };

    expect(() => decoder.decode(struct)).toThrowError(
      /does not match values length/,
    );
  });

  it('should throw when an InternedValue case is undefined', () => {
    store.addFieldPathSets([{ id: 50, fieldPathStringIds: [1] }]);
    store.addStrings([{ id: 1, value: 'item' }]);

    const struct: InternedStructDTO = {
      fieldPathSetId: 50,
      values: [{ kind: { case: undefined, value: undefined } }],
    };

    expect(() => decoder.decode(struct)).toThrowError(
      'InternedValue kind is undefined',
    );
  });
});
