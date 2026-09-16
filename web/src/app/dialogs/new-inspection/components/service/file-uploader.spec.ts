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
  FileParameterUploadResult,
} from 'src/app/services/api/file-parameter-upload-client.service';
import {
  FileUploaderStatus,
  KHIServerFileUploader,
} from 'src/app/dialogs/new-inspection/components/service/file-uploader';

describe('KHIServerFileUploader', () => {
  let uploader: KHIServerFileUploader;
  let mockUploadClient: {
    uploadFile: jasmine.Spy<
      (
        tokenId: string,
        file: File,
        options?: FileParameterUploadOptions,
      ) => Promise<FileParameterUploadResult>
    >;
  };

  beforeEach(() => {
    mockUploadClient = {
      uploadFile: jasmine.createSpy('uploadFile'),
    };

    TestBed.configureTestingModule({
      providers: [
        KHIServerFileUploader,
        {
          provide: FileParameterUploadClientService,
          useValue: mockUploadClient,
        },
      ],
    });

    uploader = TestBed.inject(KHIServerFileUploader);
  });

  it('emits done: false when onProgress reaches 100% before uploadFile promise resolves', async () => {
    let resolveUpload!: (value: FileParameterUploadResult) => void;
    const uploadPromise = new Promise<FileParameterUploadResult>((resolve) => {
      resolveUpload = resolve;
    });

    let capturedOptions: FileParameterUploadOptions | undefined;
    mockUploadClient.uploadFile.and.callFake(
      (_tokenId: string, _file: File, options?: FileParameterUploadOptions) => {
        capturedOptions = options;
        return uploadPromise;
      },
    );

    const emittedStatuses: FileUploaderStatus[] = [];
    let completed = false;

    const file = new File(['content'], 'test.txt');
    uploader.upload({ id: 'token-123' }, file).subscribe({
      next: (status) => emittedStatuses.push(status),
      complete: () => {
        completed = true;
      },
    });

    // Initial status
    expect(emittedStatuses.length).toBe(1);
    expect(emittedStatuses[0]).toEqual({
      done: false,
      completeRatio: 0,
      completeRatioUnknown: false,
    });

    // Intermediate progress
    capturedOptions?.onProgress?.(50, 100);
    expect(emittedStatuses.length).toBe(2);
    expect(emittedStatuses[1]).toEqual({
      done: false,
      completeRatio: 0.5,
      completeRatioUnknown: false,
    });

    // onProgress called with 100% before promise resolution
    capturedOptions?.onProgress?.(100, 100);
    expect(emittedStatuses.length).toBe(3);
    expect(emittedStatuses[2]).toEqual({
      done: false,
      completeRatio: 1,
      completeRatioUnknown: false,
    });
    expect(completed).toBeFalse();

    // Now resolve the promise
    resolveUpload({ fileSizeBytes: 100 });
    await uploadPromise;

    // After resolution, done: true must be emitted and the observable completed
    expect(emittedStatuses.length).toBe(4);
    expect(emittedStatuses[3]).toEqual({
      done: true,
      completeRatio: 1,
      completeRatioUnknown: false,
    });
    expect(completed).toBeTrue();
  });

  it('emits error if uploadFile promise rejects', async () => {
    let rejectUpload!: (reason: unknown) => void;
    const uploadPromise = new Promise<FileParameterUploadResult>(
      (_resolve, reject) => {
        rejectUpload = reject;
      },
    );

    mockUploadClient.uploadFile.and.returnValue(uploadPromise);

    const file = new File(['content'], 'test.txt');
    const errorPromise = new Promise<unknown>((resolve) => {
      uploader.upload({ id: 'token-123' }, file).subscribe({
        error: (err: unknown) => {
          resolve(err);
        },
      });
    });

    const expectedErr = new Error('upload failed');
    rejectUpload(expectedErr);

    const caughtError = await errorPromise;
    expect(caughtError).toBe(expectedErr);
  });
});
