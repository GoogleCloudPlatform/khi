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
import { filter, map, Observable, of } from 'rxjs';
import { UploadToken } from '../../../../common/schema/form-types';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { HttpEventType } from '@angular/common/http';

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
  private readonly backendAPI: BackendAPI = inject(BACKEND_API);

  upload(token: UploadToken, file: File): Observable<FileUploaderStatus> {
    return this.backendAPI.uploadFile(token, file).pipe(
      filter(
        (status) =>
          status.type !== HttpEventType.User &&
          status.type !== HttpEventType.DownloadProgress,
      ),
      map((status) => {
        switch (status.type) {
          case HttpEventType.Response:
            return {
              done: true,
              completeRatio: 1,
              completeRatioUnknown: false,
            };
          case HttpEventType.ResponseHeader:
          case HttpEventType.Sent:
            return {
              done: false,
              completeRatio: 0,
              completeRatioUnknown: false,
            };
          case HttpEventType.UploadProgress:
            if (status.total !== undefined) {
              // This status.total can be undefined but I don't know when it could be.
              return {
                done: status.loaded === status.total,
                completeRatio: status.loaded / status.total,
                completeRatioUnknown: false,
              };
            } else {
              return {
                done: status.loaded === status.total,
                completeRatio: 0,
                completeRatioUnknown: true,
              };
            }
          default:
            throw new Error('unknown event type' + status.type);
        }
      }),
    );
  }
}
