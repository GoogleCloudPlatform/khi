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

import {
  BufferLayoutBuilder,
  nextCapacity,
  reallocateBuffer,
  reallocateUint32Array,
} from 'src/app/store/domain/buffer-util';

describe('buffer-util', () => {
  describe('nextCapacity', () => {
    it('returns initialCapacity when currentCapacity is zero and minCapacity is small', () => {
      expect(nextCapacity(0, 10, 1024)).toBe(1024);
    });

    it('uses default initialCapacity of 1024 when omitted', () => {
      expect(nextCapacity(0, 10)).toBe(1024);
    });

    it('handles negative currentCapacity by treating it as zero', () => {
      expect(nextCapacity(-1, 10)).toBe(1024);
    });

    it('returns minCapacity when currentCapacity is zero and minCapacity exceeds initialCapacity', () => {
      expect(nextCapacity(0, 2048, 1024)).toBe(2048);
    });

    it('doubles current capacity when double exceeds minCapacity', () => {
      expect(nextCapacity(1024, 1500)).toBe(2048);
    });

    it('doubles capacity even when minCapacity is less than or equal to currentCapacity', () => {
      expect(nextCapacity(1024, 1024)).toBe(2048);
      expect(nextCapacity(1024, 500)).toBe(2048);
    });

    it('doubles repeatedly until capacity satisfies minCapacity', () => {
      expect(nextCapacity(1024, 5000)).toBe(8192);
    });
  });

  describe('reallocateBuffer', () => {
    it('allocates a new buffer and copies old data up to bytesToCopy', () => {
      const oldBuffer = new ArrayBuffer(16);
      const oldView = new Uint8Array(oldBuffer);
      for (let i = 0; i < 16; i++) {
        oldView[i] = i + 1;
      }

      const newBuffer = reallocateBuffer(oldBuffer, 32, 16);
      expect(newBuffer.byteLength).toBe(32);

      const newView = new Uint8Array(newBuffer);
      for (let i = 0; i < 16; i++) {
        expect(newView[i]).toBe(i + 1);
      }
      for (let i = 16; i < 32; i++) {
        expect(newView[i]).toBe(0);
      }
    });

    it('defaults bytesToCopy to min(old.byteLength, newByteLength) when omitted', () => {
      const oldBuffer = new ArrayBuffer(16);
      const oldView = new Uint8Array(oldBuffer);
      for (let i = 0; i < 16; i++) {
        oldView[i] = i + 1;
      }

      const expandedBuffer = reallocateBuffer(oldBuffer, 24);
      expect(expandedBuffer.byteLength).toBe(24);
      const expandedView = new Uint8Array(expandedBuffer);
      for (let i = 0; i < 16; i++) {
        expect(expandedView[i]).toBe(i + 1);
      }

      const shrunkBuffer = reallocateBuffer(oldBuffer, 8);
      expect(shrunkBuffer.byteLength).toBe(8);
      const shrunkView = new Uint8Array(shrunkBuffer);
      for (let i = 0; i < 8; i++) {
        expect(shrunkView[i]).toBe(i + 1);
      }
    });

    it('handles shrinking buffers properly', () => {
      const oldBuffer = new ArrayBuffer(16);
      const oldView = new Uint8Array(oldBuffer);
      for (let i = 0; i < 16; i++) {
        oldView[i] = i + 1;
      }

      const newBuffer = reallocateBuffer(oldBuffer, 8, 8);
      expect(newBuffer.byteLength).toBe(8);

      const newView = new Uint8Array(newBuffer);
      for (let i = 0; i < 8; i++) {
        expect(newView[i]).toBe(i + 1);
      }
    });

    it('creates a zero-filled buffer when bytesToCopy is zero', () => {
      const oldBuffer = new ArrayBuffer(16);
      new Uint8Array(oldBuffer).fill(0xff);

      const newBuffer = reallocateBuffer(oldBuffer, 16, 0);
      expect(newBuffer.byteLength).toBe(16);
      const view = new Uint8Array(newBuffer);
      for (let i = 0; i < 16; i++) {
        expect(view[i]).toBe(0);
      }
    });

    it('handles shrinking to zero byteLength', () => {
      const oldBuffer = new ArrayBuffer(16);
      new Uint8Array(oldBuffer).fill(0x42);

      const newBuffer = reallocateBuffer(oldBuffer, 0, 0);
      expect(newBuffer.byteLength).toBe(0);
    });
  });

  describe('BufferLayoutBuilder', () => {
    it('calculates sequential field offsets with default 4-byte alignment', () => {
      const builder = new BufferLayoutBuilder();
      const offset1 = builder.addField(10, 4);
      const offset2 = builder.addField(10, 4);

      expect(offset1).toBe(0);
      expect(offset2).toBe(40);
      expect(builder.totalBytes).toBe(80);
    });

    it('aligns offsets to specified alignment boundaries', () => {
      const builder = new BufferLayoutBuilder();
      // 5 items of 2 bytes = 10 bytes (offset 0..10)
      const offset16 = builder.addField(5, 2, 2);
      expect(offset16).toBe(0);
      // Next field requires 8-byte alignment, 10 aligned to 8 is 16
      const offset64 = builder.addField(2, 8, 8);
      expect(offset64).toBe(16);
      expect(builder.totalBytes).toBe(32);
    });

    it('handles capacity of 0 correctly', () => {
      const builder = new BufferLayoutBuilder();
      const offset1 = builder.addField(0, 4);
      const offset2 = builder.addField(0, 8, 8);

      expect(offset1).toBe(0);
      expect(offset2).toBe(0);
      expect(builder.totalBytes).toBe(0);
    });
  });

  describe('reallocateUint32Array', () => {
    it('reallocates Uint32Array and copies existing elements', () => {
      const oldArr = new Uint32Array([10, 20, 30]);
      const newArr = reallocateUint32Array(oldArr, 5);

      expect(newArr.length).toBe(5);
      expect(newArr[0]).toBe(10);
      expect(newArr[1]).toBe(20);
      expect(newArr[2]).toBe(30);
      expect(newArr[3]).toBe(0);
      expect(newArr[4]).toBe(0);
    });

    it('reallocates Uint32Array with specific copyLength', () => {
      const oldArr = new Uint32Array([10, 20, 30, 40]);
      const newArr = reallocateUint32Array(oldArr, 6, 2);

      expect(newArr.length).toBe(6);
      expect(newArr[0]).toBe(10);
      expect(newArr[1]).toBe(20);
      expect(newArr[2]).toBe(0);
    });

    it('handles shrinking Uint32Array', () => {
      const oldArr = new Uint32Array([10, 20, 30, 40]);
      const newArr = reallocateUint32Array(oldArr, 2);

      expect(newArr.length).toBe(2);
      expect(newArr[0]).toBe(10);
      expect(newArr[1]).toBe(20);
    });
  });
});
