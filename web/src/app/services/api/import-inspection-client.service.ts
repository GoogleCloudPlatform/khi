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

import { Injectable } from '@angular/core';
import { createClient, Client } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { ImportInspectionService } from 'src/app/generated/api/v1/import_inspection_pb';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';

/**
 * Callback function type for reporting file upload progress.
 */
export interface ImportProgressCallback {
  (uploadedBytes: number, totalBytes: number): void;
}

/**
 * Options for importing a .khi inspection file.
 */
export interface ImportFileOptions {
  /** Optional callback invoked after each uploaded chunk. */
  readonly onProgress?: ImportProgressCallback;
  /** Optional AbortSignal to cancel an in-progress upload. */
  readonly abortSignal?: AbortSignal;
  /** Optional maximum number of concurrent chunk uploads. Defaults to 4. */
  readonly maxConcurrency?: number;
}

/**
 * Result returned upon successful inspection file import and registration.
 */
export interface ImportInspectionResult {
  /** The unique ID assigned to the imported inspection. */
  readonly inspectionId: string;
  /** The display name of the inspection extracted from metadata. */
  readonly inspectionName: string;
  /** The total size in bytes of the imported inspection file. */
  readonly fileSizeBytes: number;
}

interface ChunkDescriptor {
  readonly offset: number;
  readonly length: number;
}

/**
 * Client service for uploading and importing .khi inspection files via Connect-RPC.
 */
@Injectable({
  providedIn: 'root',
})
export class ImportInspectionClientService {
  /** Default chunk size in bytes (16 MB). */
  public static readonly DEFAULT_CHUNK_SIZE_BYTES = 16 * 1024 * 1024;

  /** Default maximum number of concurrent chunk uploads. */
  public static readonly DEFAULT_MAX_CONCURRENCY = 4;

  private readonly client: Client<typeof ImportInspectionService>;

  constructor() {
    const transport = createConnectTransport({
      baseUrl: ApiPathUtil.getServerBasePath(),
    });
    this.client = createClient(ImportInspectionService, transport);
  }

  /**
   * Imports a .khi file in chunks to the backend server.
   *
   * @param file The .khi inspection file to upload.
   * @param options Optional callbacks for progress tracking, concurrency limit, and abort signaling.
   * @returns The registered inspection details upon completion.
   */
  public async importFile(
    file: File,
    options?: ImportFileOptions,
  ): Promise<ImportInspectionResult> {
    const totalSize = file.size;
    const startResponse = await this.client.startImportInspection(
      {
        fileName: file.name,
        totalSizeBytes: BigInt(totalSize),
      },
      { signal: options?.abortSignal },
    );

    const token = startResponse.importToken;
    const chunkSize =
      Number(startResponse.suggestedChunkSizeBytes) > 0
        ? Number(startResponse.suggestedChunkSizeBytes)
        : ImportInspectionClientService.DEFAULT_CHUNK_SIZE_BYTES;

    const chunks: ChunkDescriptor[] = [];
    for (let offset = 0; offset < totalSize; offset += chunkSize) {
      const length = Math.min(chunkSize, totalSize - offset);
      chunks.push({ offset, length });
    }

    const maxConcurrency = Math.max(
      1,
      options?.maxConcurrency ??
        ImportInspectionClientService.DEFAULT_MAX_CONCURRENCY,
    );

    const abortController = new AbortController();
    if (options?.abortSignal) {
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

    try {
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
            const sliceBlob = file.slice(
              chunk.offset,
              chunk.offset + chunk.length,
            );
            const buffer = await sliceBlob.arrayBuffer();
            const data = new Uint8Array(buffer);

            if (abortController.signal.aborted || firstError !== null) {
              break;
            }

            await this.client.uploadInspectionChunk(
              {
                importToken: token,
                offsetBytes: BigInt(chunk.offset),
                data,
              },
              { signal: abortController.signal },
            );

            uploadedBytes += chunk.length;
            if (options?.onProgress) {
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
      if (options?.abortSignal?.aborted) {
        throw new DOMException('Upload aborted by user', 'AbortError');
      }

      const completeResponse = await this.client.completeImportInspection(
        {
          importToken: token,
        },
        { signal: options?.abortSignal },
      );

      return {
        inspectionId: completeResponse.inspectionId,
        inspectionName: completeResponse.inspectionName,
        fileSizeBytes: Number(completeResponse.fileSizeBytes),
      };
    } catch (err) {
      try {
        await this.client.abortImportInspection({
          importToken: token,
        });
      } catch (abortErr) {
        console.warn('Failed to abort import session on server:', abortErr);
      }
      throw err;
    }
  }
}
