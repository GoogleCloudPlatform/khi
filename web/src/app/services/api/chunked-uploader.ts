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

import { CancellationError } from 'src/app/store/domain/filter/types';

/**
 * Progress callback reporting uploaded bytes and total bytes.
 */
export interface ChunkUploadProgressCallback {
  (uploadedBytes: number, totalBytes: number): void;
}

/**
 * Configuration options for executing chunked file uploads.
 */
export interface ChunkUploadExecutorOptions {
  /** The source file to upload in chunks. */
  readonly file: File;

  /** The byte size of each chunk payload. */
  readonly chunkSize: number;

  /** Maximum number of concurrent chunk uploads. Defaults to 4. */
  readonly maxConcurrency?: number;

  /** Optional abort signal to cancel in-progress chunk uploads. */
  readonly abortSignal?: AbortSignal;

  /** Optional progress callback invoked as chunks complete. */
  readonly onProgress?: ChunkUploadProgressCallback;

  /** Function that uploads a single binary chunk to the remote service. */
  readonly uploadChunk: (
    offsetBytes: number,
    data: Uint8Array,
    signal: AbortSignal,
  ) => Promise<void>;
}

interface ChunkDescriptor {
  readonly offset: number;
  readonly length: number;
}

/**
 * Default chunk size in bytes (16 MB).
 */
export const DEFAULT_CHUNK_SIZE_BYTES = 16 * 1024 * 1024;

/**
 * Default maximum number of concurrent chunk uploads.
 */
export const DEFAULT_MAX_CONCURRENCY = 4;

/**
 * Splits a file into slices and uploads them concurrently using the provided uploadChunk function.
 *
 * @param options Upload configuration including file, chunkSize, and worker callbacks.
 */
export async function executeChunkedUpload(
  options: ChunkUploadExecutorOptions,
): Promise<void> {
  const totalSize = options.file.size;
  const chunkSize =
    options.chunkSize > 0 ? options.chunkSize : DEFAULT_CHUNK_SIZE_BYTES;

  const chunks: ChunkDescriptor[] = [];
  for (let offset = 0; offset < totalSize; offset += chunkSize) {
    const length = Math.min(chunkSize, totalSize - offset);
    chunks.push({ offset, length });
  }

  // If file is empty, no chunk uploading needed
  if (chunks.length === 0) {
    return;
  }

  const maxConcurrency = Math.max(
    1,
    options.maxConcurrency ?? DEFAULT_MAX_CONCURRENCY,
  );

  const abortController = new AbortController();
  if (options.abortSignal) {
    if (options.abortSignal.aborted) {
      abortController.abort();
    } else {
      options.abortSignal.addEventListener(
        'abort',
        () => abortController.abort(),
        { once: true },
      );
    }
  }

  let nextChunkIndex = 0;
  let uploadedBytes = 0;
  let firstError: unknown = null;

  const workerCount = Math.min(maxConcurrency, chunks.length);
  const workers = Array.from({ length: workerCount }, async () => {
    while (true) {
      if (abortController.signal.aborted || firstError !== null) {
        break;
      }

      const index = nextChunkIndex++;
      if (index >= chunks.length) {
        break;
      }

      const chunk = chunks[index];
      try {
        const sliceBlob = options.file.slice(
          chunk.offset,
          chunk.offset + chunk.length,
        );
        const buffer = await sliceBlob.arrayBuffer();
        const data = new Uint8Array(buffer);

        if (abortController.signal.aborted || firstError !== null) {
          break;
        }

        await options.uploadChunk(chunk.offset, data, abortController.signal);

        uploadedBytes += chunk.length;
        if (options.onProgress) {
          options.onProgress(uploadedBytes, totalSize);
        }
      } catch (err) {
        if (firstError === null) {
          firstError = err;
          abortController.abort();
        }
        break;
      }
    }
  });

  await Promise.all(workers);

  if (firstError !== null) {
    throw firstError;
  }
  if (options.abortSignal?.aborted) {
    throw new CancellationError('Upload aborted by user');
  }
}
