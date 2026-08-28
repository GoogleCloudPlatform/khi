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
  executeChunkedUpload,
  ChunkUploadProgressCallback,
} from 'src/app/services/api/chunked-uploader';
import { CancellationError } from 'src/app/store/domain/filter/types';

describe('executeChunkedUpload', () => {
  it('should split file and upload chunks with progress updates', async () => {
    const fileContent = '1234567890abcdef'; // 16 bytes
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });

    const uploadedChunks: { offset: number; size: number }[] = [];
    const progressUpdates: number[] = [];

    const onProgress: ChunkUploadProgressCallback = (uploaded, total) => {
      progressUpdates.push(uploaded);
      expect(total).toBe(16);
    };

    await executeChunkedUpload({
      file,
      chunkSize: 5,
      maxConcurrency: 2,
      onProgress,
      uploadChunk: async (offset, data) => {
        uploadedChunks.push({ offset, size: data.byteLength });
      },
    });

    // 16 bytes with chunkSize 5 => chunks at offsets [0 (5 bytes), 5 (5 bytes), 10 (5 bytes), 15 (1 byte)]
    expect(uploadedChunks.length).toBe(4);
    uploadedChunks.sort((a, b) => a.offset - b.offset);
    expect(uploadedChunks).toEqual([
      { offset: 0, size: 5 },
      { offset: 5, size: 5 },
      { offset: 10, size: 5 },
      { offset: 15, size: 1 },
    ]);
    expect(progressUpdates[progressUpdates.length - 1]).toBe(16);
  });

  it('should handle empty file gracefully without errors', async () => {
    const file = new File([], 'empty.txt', { type: 'text/plain' });
    let called = false;

    await executeChunkedUpload({
      file,
      chunkSize: 10,
      uploadChunk: async () => {
        called = true;
      },
    });

    expect(called).toBeFalse();
  });

  it('should abort upload when uploadChunk throws an error', async () => {
    const fileContent = '1234567890abcdef';
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });

    await expectAsync(
      executeChunkedUpload({
        file,
        chunkSize: 4,
        uploadChunk: async (offset) => {
          if (offset === 4) {
            throw new Error('chunk upload failed');
          }
        },
      }),
    ).toBeRejectedWithError('chunk upload failed');
  });

  it('should retry chunk on transient error and complete successfully', async () => {
    const fileContent = '1234567890abcdef'; // 16 bytes, 4 chunks of 4 bytes
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });
    let failedOnce = false;
    let offset4Calls = 0;

    await executeChunkedUpload({
      file,
      chunkSize: 4,
      maxRetriesPerChunk: 2,
      uploadChunk: async (offset) => {
        if (offset === 4) {
          offset4Calls++;
          if (!failedOnce) {
            failedOnce = true;
            throw new Error('503 Service Unavailable');
          }
        }
      },
    });

    expect(failedOnce).toBeTrue();
    expect(offset4Calls).toBe(2);
  });

  it('should fail when chunk retries are exhausted', async () => {
    const fileContent = '1234567890abcdef';
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });
    let callCount = 0;

    await expectAsync(
      executeChunkedUpload({
        file,
        chunkSize: 4,
        maxRetriesPerChunk: 2,
        uploadChunk: async (offset) => {
          if (offset === 0) {
            callCount++;
            throw new Error('502 Bad Gateway');
          }
        },
      }),
    ).toBeRejectedWithError('502 Bad Gateway');

    // 1 initial attempt + 2 retries = 3 calls
    expect(callCount).toBe(3);
  });

  it('should fail immediately on non-retryable error without retrying', async () => {
    const fileContent = '1234567890abcdef';
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });
    let callCount = 0;

    await expectAsync(
      executeChunkedUpload({
        file,
        chunkSize: 4,
        maxRetriesPerChunk: 3,
        uploadChunk: async (offset) => {
          if (offset === 0) {
            callCount++;
            throw new Error('400 Bad Request');
          }
        },
      }),
    ).toBeRejectedWithError('400 Bad Request');

    expect(callCount).toBe(1);
  });

  it('should abort upload when abortSignal is triggered', async () => {
    const fileContent = '1234567890abcdef';
    const file = new File([fileContent], 'test.txt', { type: 'text/plain' });

    const controller = new AbortController();

    await expectAsync(
      executeChunkedUpload({
        file,
        chunkSize: 4,
        abortSignal: controller.signal,
        uploadChunk: async (offset) => {
          if (offset === 0) {
            controller.abort();
          }
        },
      }),
    ).toBeRejectedWith(jasmine.any(CancellationError));
  });
});
