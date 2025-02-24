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

import { InjectionToken } from '@angular/core';
import { Observable, of } from 'rxjs';

/**
 * The ID of token given from the backend to identify uploaded files.
 */
export interface UploadToken {
  id: string;
}

/**
 * The types of UploadStatus given from the backend.
 */
export enum UploadStatus {
  Waiting = 0,
  Uploading = 1,
  Verifying = 2,
  Done = 3,
}

/**
 * Type for the status reported from the uploader.
 */
export interface FileUploaderStatus {
  done: boolean;
  completeRatio: number;
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
    });

  public statusProvider: () => Observable<FileUploaderStatus> =
    MockFileUploader.MOCK_COMPLETED_UPLOAD_STATUS_PROVIDER;

  upload(): Observable<FileUploaderStatus> {
    return this.statusProvider();
  }
}
