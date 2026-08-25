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

import { Injectable, inject } from '@angular/core';
import { Client } from '@connectrpc/connect';
import { ImportInspectionService } from 'src/app/generated/api/v1/import_inspection_pb';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import {
  executeChunkedUpload,
  ChunkUploadProgressCallback,
  DEFAULT_CHUNK_SIZE_BYTES,
  DEFAULT_MAX_CONCURRENCY,
} from 'src/app/services/api/chunked-uploader';

/**
 * Callback function type for reporting file upload progress.
 */
export type ImportProgressCallback = ChunkUploadProgressCallback;

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

/**
 * Client service for uploading and importing .khi inspection files via Connect-RPC.
 */
@Injectable({
  providedIn: 'root',
})
export class ImportInspectionClientService {
  /** Default chunk size in bytes (16 MB). */
  public static readonly DEFAULT_CHUNK_SIZE_BYTES = DEFAULT_CHUNK_SIZE_BYTES;

  /** Default maximum number of concurrent chunk uploads. */
  public static readonly DEFAULT_MAX_CONCURRENCY = DEFAULT_MAX_CONCURRENCY;

  private readonly connectClient = inject(ConnectClientService);
  private readonly client: Client<typeof ImportInspectionService> =
    this.connectClient.importInspectionClient;

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

    try {
      await executeChunkedUpload({
        file,
        chunkSize,
        maxConcurrency: options?.maxConcurrency,
        abortSignal: options?.abortSignal,
        onProgress: options?.onProgress,
        uploadChunk: async (offsetBytes, data, signal) => {
          await this.client.uploadInspectionChunk(
            {
              importToken: token,
              offsetBytes: BigInt(offsetBytes),
              data,
            },
            { signal },
          );
        },
      });

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
