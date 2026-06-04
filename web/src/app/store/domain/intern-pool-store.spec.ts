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
    store = InternPoolStore.create();
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

  it('should split buffer if the string size exceeds maxBufferSize', () => {
    const smallStore = InternPoolStore.create(10);
    smallStore.addStrings([
      { id: 1, value: 'abcdefgh' }, // 8 bytes -> fits in 1st buffer
      { id: 2, value: 'ijklmnop' }, // 8 bytes -> exceeds remaining 2 bytes, goes to 2nd buffer
      { id: 3, value: 'qrstuvwxyz12345' }, // 15 bytes -> exceeds 10 bytes maxBufferSize, allocated standalone
      { id: 4, value: 'abc' }, // 3 bytes -> fits in next buffer
    ]);

    expect(smallStore.getString(1)).toBe('abcdefgh');
    expect(smallStore.getString(2)).toBe('ijklmnop');
    expect(smallStore.getString(3)).toBe('qrstuvwxyz12345');
    expect(smallStore.getString(4)).toBe('abc');
  });

  it('should resize metadata TypedArrays when string ID is large', () => {
    store.addStrings([{ id: 2000, value: 'large-id-string' }]);

    expect(store.getString(2000)).toBe('large-id-string');
  });

  it('should allow reconstructing string values from getSharedData() buffers', () => {
    store.addStrings([
      { id: 10, value: 'shared-string-1' },
      { id: 20, value: 'shared-string-2' },
    ]);

    const sharedData = store.getSharedData();

    const localIndices = new Uint16Array(
      sharedData.metadataSab,
      0,
      sharedData.capacity,
    );
    const localOffsets = new Uint32Array(
      sharedData.metadataSab,
      sharedData.capacity * 2,
      sharedData.capacity,
    );
    const localLengths = new Uint32Array(
      sharedData.metadataSab,
      sharedData.capacity * 6,
      sharedData.capacity,
    );

    const index10 = localIndices[10] - 1;
    const offset10 = localOffsets[10];
    const length10 = localLengths[10];

    const buffer10 = new Uint8Array(sharedData.bufferSabs[index10]);
    const bytes10 = buffer10.subarray(offset10, offset10 + length10);
    const nonSharedBytes10 = new Uint8Array(length10);
    nonSharedBytes10.set(bytes10);
    const decoder = new TextDecoder();

    expect(decoder.decode(nonSharedBytes10)).toBe('shared-string-1');

    const index20 = localIndices[20] - 1;
    const offset20 = localOffsets[20];
    const length20 = localLengths[20];

    const buffer20 = new Uint8Array(sharedData.bufferSabs[index20]);
    const bytes20 = buffer20.subarray(offset20, offset20 + length20);
    const nonSharedBytes20 = new Uint8Array(length20);
    nonSharedBytes20.set(bytes20);

    expect(decoder.decode(nonSharedBytes20)).toBe('shared-string-2');
  });

  it('should be transmissible via postMessage and decodable inside a WebWorker', (done) => {
    store.addStrings([{ id: 10, value: 'worker-shared-string' }]);

    const sharedData = store.getSharedData();

    const workerCode = `
      self.onmessage = function(e) {
        try {
          const sharedData = e.data;
          const localIndices = new Uint16Array(sharedData.metadataSab, 0, sharedData.capacity);
          const localOffsets = new Uint32Array(sharedData.metadataSab, sharedData.capacity * 2, sharedData.capacity);
          const localLengths = new Uint32Array(sharedData.metadataSab, sharedData.capacity * 6, sharedData.capacity);

          const index = localIndices[10] - 1;
          const offset = localOffsets[10];
          const length = localLengths[10];

          const buffer = new Uint8Array(sharedData.bufferSabs[index]);
          const bytes = buffer.subarray(offset, offset + length);
          const nonSharedBytes = new Uint8Array(length);
          nonSharedBytes.set(bytes);
          const decoded = new TextDecoder().decode(nonSharedBytes);

          self.postMessage({ success: true, decoded });
        } catch (err) {
          self.postMessage({ success: false, error: err.toString() });
        }
      };
    `;
    const blob = new Blob([workerCode], { type: 'application/javascript' });
    const worker = new Worker(URL.createObjectURL(blob));

    worker.onmessage = (e) => {
      const result = e.data;
      if (result.success) {
        expect(result.decoded).toBe('worker-shared-string');
      } else {
        fail('Worker failed with error: ' + result.error);
      }
      worker.terminate();
      done();
    };

    worker.postMessage(sharedData);
  });

  it('should initialize successfully from sharedData using fromSharedData()', () => {
    store.addStrings([
      { id: 10, value: 'shared-string-1' },
      { id: 20, value: 'shared-string-2' },
    ]);
    store.addFieldPathSets([{ id: 100, fieldPathStringIds: [10, 20] }]);

    const sharedData = store.getSharedData();
    const sharedStore = InternPoolStore.fromSharedData(sharedData);

    expect(sharedStore.getString(10)).toBe('shared-string-1');
    expect(sharedStore.getString(20)).toBe('shared-string-2');
    expect(sharedStore.getFieldPathSet(100)).toEqual([
      'shared-string-1',
      'shared-string-2',
    ]);
  });

  it('should throw an error on write attempts on a read-only instance created from sharedData', () => {
    store.addStrings([{ id: 10, value: 'shared-string' }]);
    const sharedData = store.getSharedData();
    const sharedStore = InternPoolStore.fromSharedData(sharedData);

    expect(() => {
      sharedStore.addStrings([{ id: 11, value: 'fail' }]);
    }).toThrowError('Cannot write to a shared read-only InternPoolStore');

    expect(() => {
      sharedStore.addFieldPathSets([{ id: 200, fieldPathStringIds: [10] }]);
    }).toThrowError('Cannot write to a shared read-only InternPoolStore');
  });

  describe('ArrayBuffer fallback when SharedArrayBuffer is unsupported', () => {
    let originalSharedArrayBuffer: typeof SharedArrayBuffer | undefined;

    beforeEach(() => {
      originalSharedArrayBuffer = SharedArrayBuffer;
      (
        globalThis as unknown as Record<
          string,
          typeof SharedArrayBuffer | undefined
        >
      )['SharedArrayBuffer'] = undefined;
    });

    afterEach(() => {
      (
        globalThis as unknown as Record<
          string,
          typeof SharedArrayBuffer | undefined
        >
      )['SharedArrayBuffer'] = originalSharedArrayBuffer;
    });

    it('should allocate ArrayBuffer instead of SharedArrayBuffer and perform operations successfully', () => {
      const fallbackStore = InternPoolStore.create();
      fallbackStore.addStrings([
        { id: 10, value: 'fallback-string-1' },
        { id: 20, value: 'fallback-string-2' },
      ]);

      expect(fallbackStore.getString(10)).toBe('fallback-string-1');
      expect(fallbackStore.getString(20)).toBe('fallback-string-2');

      const sharedData = fallbackStore.getSharedData();
      expect(sharedData.metadataSab instanceof ArrayBuffer).toBeTrue();
      expect(sharedData.bufferSabs[0] instanceof ArrayBuffer).toBeTrue();

      const reconstructedStore = InternPoolStore.fromSharedData(sharedData);
      expect(reconstructedStore.getString(10)).toBe('fallback-string-1');
    });
  });
});
