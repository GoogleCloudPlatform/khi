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

import { TestBed } from '@angular/core/testing';
import {
  FileParameterUploadClientService,
  FileParameterUploadOptions,
} from 'src/app/services/api/file-parameter-upload-client.service';
import { Client } from '@connectrpc/connect';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { FileParameterUploadService } from 'src/app/generated/api/v1/file_parameter_upload_pb';
import { environment } from 'src/environments/environment';

describe('FileParameterUploadClientService', () => {
  let service: FileParameterUploadClientService;
  let mockClient: jasmine.SpyObj<Client<typeof FileParameterUploadService>>;
  let mockConnectClient: jasmine.SpyObj<ConnectClientService>;

  beforeEach(() => {
    mockClient = jasmine.createSpyObj<
      Client<typeof FileParameterUploadService>
    >('FileParameterUploadClient', [
      'startFileUpload',
      'uploadFileChunk',
      'completeFileUpload',
      'abortFileUpload',
    ]);
    mockConnectClient = jasmine.createSpyObj<ConnectClientService>(
      'ConnectClientService',
      [],
      {
        fileParameterUploadClient: mockClient,
      },
    );

    TestBed.configureTestingModule({
      providers: [
        FileParameterUploadClientService,
        { provide: ConnectClientService, useValue: mockConnectClient },
      ],
    });

    service = TestBed.inject(FileParameterUploadClientService);
  });

  it('uploads a file in chunks and returns result on complete', async () => {
    mockClient.startFileUpload.and.returnValue(
      Promise.resolve({
        sessionToken: 'session-123',
        suggestedChunkSizeBytes: BigInt(5),
      } as unknown as Awaited<ReturnType<typeof mockClient.startFileUpload>>),
    );

    mockClient.uploadFileChunk.and.returnValue(
      Promise.resolve({
        totalReceivedBytes: BigInt(5),
      } as unknown as Awaited<ReturnType<typeof mockClient.uploadFileChunk>>),
    );

    mockClient.completeFileUpload.and.returnValue(
      Promise.resolve({
        fileSizeBytes: BigInt(12),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.completeFileUpload>
      >),
    );

    const progressCalls: { uploaded: number; total: number }[] = [];
    const options: FileParameterUploadOptions = {
      maxConcurrency: 1,
      onProgress: (uploaded, total) => {
        progressCalls.push({ uploaded, total });
      },
    };

    const dummyContent = new Uint8Array([
      1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
    ]);
    const file = new File([dummyContent], 'data.log', {
      type: 'application/octet-stream',
    });

    const result = await service.uploadFile('token-xyz', file, options);

    expect(mockClient.startFileUpload).toHaveBeenCalledWith(
      {
        uploadTokenId: 'token-xyz',
        fileName: 'data.log',
        totalSizeBytes: BigInt(12),
      },
      { signal: undefined },
    );

    expect(mockClient.uploadFileChunk).toHaveBeenCalledTimes(3);
    expect(progressCalls).toEqual([
      { uploaded: 5, total: 12 },
      { uploaded: 10, total: 12 },
      { uploaded: 12, total: 12 },
    ]);

    expect(mockClient.completeFileUpload).toHaveBeenCalledWith(
      {
        sessionToken: 'session-123',
      },
      { signal: undefined },
    );

    expect(result).toEqual({
      fileSizeBytes: 12,
    });
  });

  it('aborts upload and invokes abortFileUpload on server when chunk upload fails', async () => {
    mockClient.startFileUpload.and.returnValue(
      Promise.resolve({
        sessionToken: 'session-error',
        suggestedChunkSizeBytes: BigInt(10),
      } as unknown as Awaited<ReturnType<typeof mockClient.startFileUpload>>),
    );

    mockClient.uploadFileChunk.and.returnValue(
      Promise.reject(new Error('Network failure')),
    );

    mockClient.abortFileUpload.and.returnValue(
      Promise.resolve({
        aborted: true,
      } as unknown as Awaited<ReturnType<typeof mockClient.abortFileUpload>>),
    );

    const file = new File([new Uint8Array(10)], 'error.log');

    await expectAsync(
      service.uploadFile('token-err', file),
    ).toBeRejectedWithError(/Network failure/);

    expect(mockClient.abortFileUpload).toHaveBeenCalledWith({
      sessionToken: 'session-error',
    });
  });

  it('overrides server suggestedChunkSizeBytes when environment.upload is configured', async () => {
    const originalUpload = environment.upload;
    try {
      (environment as { upload: typeof originalUpload }).upload = {
        chunkSizeBytes: 4,
        maxConcurrency: 2,
      };

      mockClient.startFileUpload.and.returnValue(
        Promise.resolve({
          sessionToken: 'session-env',
          suggestedChunkSizeBytes: BigInt(20),
        } as unknown as Awaited<ReturnType<typeof mockClient.startFileUpload>>),
      );

      mockClient.uploadFileChunk.and.returnValue(
        Promise.resolve({
          totalReceivedBytes: BigInt(4),
        } as unknown as Awaited<ReturnType<typeof mockClient.uploadFileChunk>>),
      );

      mockClient.completeFileUpload.and.returnValue(
        Promise.resolve({
          fileSizeBytes: BigInt(8),
        } as unknown as Awaited<
          ReturnType<typeof mockClient.completeFileUpload>
        >),
      );

      const file = new File([new Uint8Array(8)], 'env-param.log');
      const result = await service.uploadFile('token-env', file);

      // 8 bytes with 4 bytes chunk size => 2 chunks (even though server suggested 20)
      expect(mockClient.uploadFileChunk).toHaveBeenCalledTimes(2);
      expect(result.fileSizeBytes).toBe(8);
    } finally {
      (environment as { upload: typeof originalUpload }).upload =
        originalUpload;
    }
  });
});
