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
import { FileParameterUploadService } from 'src/app/generated/api/v1/file_parameter_upload_pb';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import {
  executeChunkedUpload,
  ChunkUploadProgressCallback,
} from 'src/app/services/api/chunked-uploader';
import { resolveUploadConfig } from 'src/app/services/api/upload-config-resolver';

/**
 * Options for uploading a file parameter in chunks.
 */
export interface FileParameterUploadOptions {
  /** Optional callback invoked after each uploaded chunk. */
  readonly onProgress?: ChunkUploadProgressCallback;

  /** Optional AbortSignal to cancel an in-progress upload. */
  readonly abortSignal?: AbortSignal;

  /** Optional maximum number of concurrent chunk uploads. */
  readonly maxConcurrency?: number;
}

/**
 * Result returned upon successful file parameter upload completion.
 */
export interface FileParameterUploadResult {
  /** The validated file size in bytes stored on the backend. */
  readonly fileSizeBytes: number;
}

/**
 * Client service for uploading form parameter files via Connect-RPC in chunks.
 */
@Injectable({
  providedIn: 'root',
})
export class FileParameterUploadClientService {
  private readonly connectClient = inject(ConnectClientService);
  private readonly client: Client<typeof FileParameterUploadService> =
    this.connectClient.fileParameterUploadClient;

  /**
   * Uploads a file parameter to the backend using chunked transfer.
   *
   * @param uploadTokenId The token ID assigned to the inspection form file field.
   * @param file The file to upload.
   * @param options Optional configuration for progress callback, concurrency, and cancellation.
   * @returns Details of the uploaded file on the backend.
   */
  public async uploadFile(
    uploadTokenId: string,
    file: File,
    options?: FileParameterUploadOptions,
  ): Promise<FileParameterUploadResult> {
    const totalSize = file.size;
    const startResponse = await this.client.startFileUpload(
      {
        uploadTokenId,
        fileName: file.name,
        totalSizeBytes: BigInt(totalSize),
      },
      { signal: options?.abortSignal },
    );

    const sessionToken = startResponse.sessionToken;
    const uploadConfig = resolveUploadConfig({
      suggestedChunkSizeBytes: Number(startResponse.suggestedChunkSizeBytes),
      callerMaxConcurrency: options?.maxConcurrency,
    });

    try {
      await executeChunkedUpload({
        file,
        chunkSize: uploadConfig.chunkSize,
        maxConcurrency: uploadConfig.maxConcurrency,
        abortSignal: options?.abortSignal,
        onProgress: options?.onProgress,
        uploadChunk: async (offsetBytes, data, signal) => {
          await this.client.uploadFileChunk(
            {
              sessionToken,
              offsetBytes: BigInt(offsetBytes),
              data,
            },
            { signal },
          );
        },
      });

      const completeResponse = await this.client.completeFileUpload(
        {
          sessionToken,
        },
        { signal: options?.abortSignal },
      );

      return {
        fileSizeBytes: Number(completeResponse.fileSizeBytes),
      };
    } catch (err) {
      // Best-effort abort if the session failed or was canceled
      try {
        await this.client.abortFileUpload({ sessionToken });
      } catch {
        // Ignore secondary abort failures
      }
      throw err;
    }
  }
}
