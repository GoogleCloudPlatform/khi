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
  ImportInspectionClientService,
  ImportFileOptions,
} from 'src/app/services/api/import-inspection-client.service';
import { Client } from '@connectrpc/connect';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { ImportInspectionService } from 'src/app/generated/api/v1/import_inspection_pb';
import { environment } from 'src/environments/environment';

describe('ImportInspectionClientService', () => {
  let service: ImportInspectionClientService;
  let mockClient: jasmine.SpyObj<Client<typeof ImportInspectionService>>;
  let mockConnectClient: jasmine.SpyObj<ConnectClientService>;

  beforeEach(() => {
    mockClient = jasmine.createSpyObj<Client<typeof ImportInspectionService>>(
      'ImportInspectionClient',
      [
        'startImportInspection',
        'uploadInspectionChunk',
        'completeImportInspection',
        'abortImportInspection',
      ],
    );
    mockConnectClient = jasmine.createSpyObj<ConnectClientService>(
      'ConnectClientService',
      [],
      {
        importInspectionClient: mockClient,
      },
    );

    TestBed.configureTestingModule({
      providers: [
        ImportInspectionClientService,
        { provide: ConnectClientService, useValue: mockConnectClient },
      ],
    });

    service = TestBed.inject(ImportInspectionClientService);
  });

  it('uploads a file in chunks and returns inspection details on complete', async () => {
    mockClient.startImportInspection.and.returnValue(
      Promise.resolve({
        importToken: 'test-token-123',
        suggestedChunkSizeBytes: BigInt(5),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.startImportInspection>
      >),
    );

    mockClient.uploadInspectionChunk.and.returnValue(
      Promise.resolve({
        totalReceivedBytes: BigInt(5),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.uploadInspectionChunk>
      >),
    );

    mockClient.completeImportInspection.and.returnValue(
      Promise.resolve({
        inspectionId: 'inspection-abc',
        inspectionName: 'Test Cluster Investigation',
        fileSizeBytes: BigInt(12),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.completeImportInspection>
      >),
    );

    const progressCalls: { uploaded: number; total: number }[] = [];
    const options: ImportFileOptions = {
      maxConcurrency: 1,
      onProgress: (uploaded, total) => {
        progressCalls.push({ uploaded, total });
      },
    };

    const dummyContent = new Uint8Array([
      1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
    ]);
    const file = new File([dummyContent], 'cluster.khi', {
      type: 'application/octet-stream',
    });

    const result = await service.importFile(file, options);

    expect(mockClient.startImportInspection).toHaveBeenCalledWith(
      {
        fileName: 'cluster.khi',
        totalSizeBytes: BigInt(12),
      },
      { signal: undefined },
    );

    // 12 bytes with 5 bytes chunk size => 3 chunks (0-5, 5-10, 10-12)
    expect(mockClient.uploadInspectionChunk).toHaveBeenCalledTimes(3);
    expect(progressCalls).toEqual([
      { uploaded: 5, total: 12 },
      { uploaded: 10, total: 12 },
      { uploaded: 12, total: 12 },
    ]);

    expect(mockClient.completeImportInspection).toHaveBeenCalledWith(
      {
        importToken: 'test-token-123',
      },
      { signal: undefined },
    );

    expect(result).toEqual({
      inspectionId: 'inspection-abc',
      inspectionName: 'Test Cluster Investigation',
      fileSizeBytes: 12,
    });
  });

  it('uploads chunks concurrently up to maxConcurrency', async () => {
    mockClient.startImportInspection.and.returnValue(
      Promise.resolve({
        importToken: 'test-token-parallel',
        suggestedChunkSizeBytes: BigInt(2),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.startImportInspection>
      >),
    );

    let activeUploads = 0;
    let maxActiveUploadsObserved = 0;

    mockClient.uploadInspectionChunk.and.callFake(async () => {
      activeUploads++;
      maxActiveUploadsObserved = Math.max(
        maxActiveUploadsObserved,
        activeUploads,
      );
      // Small async delay to test concurrency overlap
      await new Promise((resolve) => setTimeout(resolve, 10));
      activeUploads--;
      return { totalReceivedBytes: BigInt(2) } as unknown as Awaited<
        ReturnType<typeof mockClient.uploadInspectionChunk>
      >;
    });

    mockClient.completeImportInspection.and.returnValue(
      Promise.resolve({
        inspectionId: 'inspection-parallel',
        inspectionName: 'Parallel Test',
        fileSizeBytes: BigInt(10),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.completeImportInspection>
      >),
    );

    const file = new File([new Uint8Array(10)], 'parallel.khi');
    const result = await service.importFile(file, { maxConcurrency: 3 });

    expect(mockClient.uploadInspectionChunk).toHaveBeenCalledTimes(5);
    expect(maxActiveUploadsObserved).toBe(3);
    expect(result.inspectionId).toBe('inspection-parallel');
  });

  it('aborts upload and invokes abortImportInspection when upload fails', async () => {
    mockClient.startImportInspection.and.returnValue(
      Promise.resolve({
        importToken: 'token-error',
        suggestedChunkSizeBytes: BigInt(10),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.startImportInspection>
      >),
    );

    mockClient.uploadInspectionChunk.and.returnValue(
      Promise.reject(new Error('Network error during upload')),
    );

    mockClient.abortImportInspection.and.returnValue(
      Promise.resolve({
        aborted: true,
      } as unknown as Awaited<
        ReturnType<typeof mockClient.abortImportInspection>
      >),
    );

    const file = new File([new Uint8Array(10)], 'error.khi');

    await expectAsync(service.importFile(file)).toBeRejectedWithError(
      /Network error/,
    );
    expect(mockClient.abortImportInspection).toHaveBeenCalledWith({
      importToken: 'token-error',
    });
  });

  it('handles user-triggered AbortSignal properly', async () => {
    mockClient.startImportInspection.and.returnValue(
      Promise.resolve({
        importToken: 'token-abort',
        suggestedChunkSizeBytes: BigInt(2),
      } as unknown as Awaited<
        ReturnType<typeof mockClient.startImportInspection>
      >),
    );

    const controller = new AbortController();

    mockClient.uploadInspectionChunk.and.callFake(async () => {
      controller.abort();
      return { totalReceivedBytes: BigInt(2) } as unknown as Awaited<
        ReturnType<typeof mockClient.uploadInspectionChunk>
      >;
    });

    mockClient.abortImportInspection.and.returnValue(
      Promise.resolve({
        aborted: true,
      } as unknown as Awaited<
        ReturnType<typeof mockClient.abortImportInspection>
      >),
    );

    const file = new File([new Uint8Array(10)], 'abort.khi');

    await expectAsync(
      service.importFile(file, { abortSignal: controller.signal }),
    ).toBeRejected();
    expect(mockClient.abortImportInspection).toHaveBeenCalledWith({
      importToken: 'token-abort',
    });
  });

  it('overrides server suggestedChunkSizeBytes when environment.upload is configured', async () => {
    const originalUpload = environment.upload;
    try {
      (environment as { upload: typeof originalUpload }).upload = {
        chunkSizeBytes: 3,
        maxConcurrency: 2,
      };

      mockClient.startImportInspection.and.returnValue(
        Promise.resolve({
          importToken: 'test-token-env',
          suggestedChunkSizeBytes: BigInt(10),
        } as unknown as Awaited<
          ReturnType<typeof mockClient.startImportInspection>
        >),
      );

      mockClient.uploadInspectionChunk.and.returnValue(
        Promise.resolve({
          totalReceivedBytes: BigInt(3),
        } as unknown as Awaited<
          ReturnType<typeof mockClient.uploadInspectionChunk>
        >),
      );

      mockClient.completeImportInspection.and.returnValue(
        Promise.resolve({
          inspectionId: 'inspection-env',
          inspectionName: 'Env Test',
          fileSizeBytes: BigInt(9),
        } as unknown as Awaited<
          ReturnType<typeof mockClient.completeImportInspection>
        >),
      );

      const file = new File([new Uint8Array(9)], 'env.khi');
      const result = await service.importFile(file);

      // 9 bytes with 3 bytes chunk size => 3 chunks (even though server suggested 10)
      expect(mockClient.uploadInspectionChunk).toHaveBeenCalledTimes(3);
      expect(result.inspectionId).toBe('inspection-env');
    } finally {
      (environment as { upload: typeof originalUpload }).upload =
        originalUpload;
    }
  });
});
