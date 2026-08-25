/**
 * Copyright 2025 Google LLC
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

import { inject, InjectionToken } from '@angular/core';
import { Observable, of } from 'rxjs';
import { UploadToken } from '../../../../common/schema/form-types';
import { FileParameterUploadClientService } from 'src/app/services/api/file-parameter-upload-client.service';

/**
 * Type for the status reported from the uploader.
 */
export interface FileUploaderStatus {
  done: boolean;
  completeRatio: number;
  completeRatioUnknown: boolean;
}

/**
 * InjectionToken to receive the implementation of FileUploader.
 */
export const FILE_UPLOADER = new InjectionToken<FileUploader>('FILE_UPLOADER');

/**
 * FileUploader provides functionality of uploading file to the given UploadToken.
 */
export interface FileUploader {
  /**
   * Upload a file tied with the UploadToken.
   */
  upload(token: UploadToken, file: File): Observable<FileUploaderStatus>;
}

/**
 * A mock implementation of FileUploader.
 */
export class MockFileUploader implements FileUploader {
  public static readonly MOCK_COMPLETED_UPLOAD_STATUS_PROVIDER = () =>
    of({
      done: true,
      completeRatio: 1,
      completeRatioUnknown: false,
    });

  public statusProvider: () => Observable<FileUploaderStatus> =
    MockFileUploader.MOCK_COMPLETED_UPLOAD_STATUS_PROVIDER;

  upload(): Observable<FileUploaderStatus> {
    return this.statusProvider();
  }
}

/**
 * An implementation of the file uploader to the KHI server.
 */
export class KHIServerFileUploader implements FileUploader {
  private readonly uploadClient: FileParameterUploadClientService = inject(
    FileParameterUploadClientService,
  );

  upload(token: UploadToken, file: File): Observable<FileUploaderStatus> {
    return new Observable<FileUploaderStatus>((subscriber) => {
      const abortController = new AbortController();

      subscriber.next({
        done: false,
        completeRatio: 0,
        completeRatioUnknown: false,
      });

      this.uploadClient
        .uploadFile(token.id, file, {
          abortSignal: abortController.signal,
          onProgress: (uploadedBytes, totalBytes) => {
            if (totalBytes > 0) {
              subscriber.next({
                done: uploadedBytes === totalBytes,
                completeRatio: uploadedBytes / totalBytes,
                completeRatioUnknown: false,
              });
            } else {
              subscriber.next({
                done: false,
                completeRatio: 0,
                completeRatioUnknown: true,
              });
            }
          },
        })
        .then(() => {
          subscriber.next({
            done: true,
            completeRatio: 1,
            completeRatioUnknown: false,
          });
          subscriber.complete();
        })
        .catch((err: unknown) => {
          subscriber.error(err);
        });

      return () => {
        abortController.abort();
      };
    });
  }
}
