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

import { CommonModule } from '@angular/common';
import {
  Component,
  ElementRef,
  inject,
  input,
  OnDestroy,
  signal,
  ViewChild,
} from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { FILE_UPLOADER } from './service/file-uploader';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import {
  FileParameterFormField,
  UploadStatus,
} from 'src/app/common/schema/form-types';
import { ParameterHeaderComponent } from './parameter-header.component';
import { ParameterHintComponent } from './parameter-hint.component';
import { PARAMETER_STORE } from './service/parameter-store';
import { MatTooltipModule } from '@angular/material/tooltip';
import { interval, Subject, takeUntil, takeWhile } from 'rxjs';

@Component({
  selector: 'khi-new-inspection-file-parameter',
  templateUrl: './file-parameter.component.html',
  styleUrls: ['./file-parameter.component.sass'],

  imports: [
    CommonModule,
    MatFormFieldModule,
    MatTooltipModule,
    ReactiveFormsModule,
    MatIconModule,
    MatButtonModule,
    MatSnackBarModule,
    MatProgressSpinnerModule,
    ParameterHeaderComponent,
    ParameterHintComponent,
  ],
})
export class FileParameterComponent implements OnDestroy {
  /**
   * The interval to attempt to retrieve the form status from backend.
   */
  static readonly FORM_STATUS_POLLING_INTERVAL_MS = 500;

  readonly UploadStatus = UploadStatus;
  /**
   * The setting of this file type form field.
   */
  parameter = input.required<FileParameterFormField>();

  /**
   * The state if currently selected file is uploaded or not.
   * This parameter is a status only for the currently selected file, `uploadStatus` is a status for this field.
   * This parameter will be false even if `uploadStatus` is completed when user successfully uploaded their file on the server but opened a new file on the form.
   */
  isSelectedFileUploaded = signal(true);

  isSelectedFileUploading = signal(false);

  /**
   * The status if a file is dragged over the file dropping area.
   */
  fileDraggingOverArea = signal(false);

  /**
   * The ratio of file size completed upload.
   * This ratio may be undefined only when the file size information is not available.
   */
  uploadRatio = signal<number | undefined>(0);

  /**
   * The filename uploaded or will be uploaded on this field.
   * This state directly hold by FileUploadComponent and not used except for users to know which they uploaded.
   */
  filename = signal('');

  @ViewChild('fileInput')
  fileInput!: ElementRef<HTMLInputElement>;

  selectedFile: File | null = null;

  private formStoreRefreshCancel = new Subject();

  private snackBar = inject(MatSnackBar);

  private uploader = inject(FILE_UPLOADER);

  private store = inject(PARAMETER_STORE);

  /**
   * Event handler of clicking the drop area.
   */
  onClickFileDialogOpen() {
    this.fileInput.nativeElement.click();
  }

  /**
   * Eventhandler for change event of the hidden file input opened the file dialog.
   */
  onSelectedFileChangedFromDialog() {
    this.processReceivedFileInfo(
      Array.from(this.fileInput.nativeElement.files ?? []),
    );
  }

  /**
   * Eventhandler for dragenter of the dropping area.
   */
  onFileDragEnter(e: DragEvent) {
    this.fileDraggingOverArea.set(true);
    e.preventDefault();
  }

  /**
   * Eventhandler for dragleave of the dropping area.
   */
  onFileDragLeave() {
    this.fileDraggingOverArea.set(false);
  }

  /**
   * Eventhandler for dragover of the dropping area.
   */
  onFileDragOver(e: DragEvent) {
    e.preventDefault(); // needs preventDefault() in dragover and dragenter not to open the file directly with the browser page.
  }

  /**
   * Eventhandler for drop of the dropping area.
   */
  onFileDrop(e: DragEvent) {
    e.preventDefault();
    e.stopImmediatePropagation();
    this.fileDraggingOverArea.set(false);
    this.processReceivedFileInfo(Array.from(e.dataTransfer?.files ?? []));
  }

  /**
   * Eventhandler for the upload button.
   */
  onClickUploadButton() {
    if (this.selectedFile === null) {
      return;
    }
    this.isSelectedFileUploading.set(true);
    this.uploader
      .upload(this.parameter().token, this.selectedFile)
      .subscribe((status) => {
        if (status.completeRatioUnknown) {
          this.uploadRatio.set(undefined);
        } else {
          this.uploadRatio.set(status.completeRatio);
        }

        this.requestStoreRefresh();
        if (status.done) {
          this.isSelectedFileUploading.set(false);
          this.isSelectedFileUploaded.set(true);
          this.monitorRefreshingFormStoreWhileVerification();
        }
      });
  }

  processReceivedFileInfo(files: File[]) {
    if (files.length > 1) {
      this.snackBar.open('2 or more files are specified at once.');
    }
    const file = files[0];
    this.filename.set(file.name);
    this.isSelectedFileUploaded.set(false);
    this.selectedFile = file;
  }

  /**
   * Request refreshing the store status forcibly.
   * File form doesn't store meaningful parameter into the parameter store because it uploads file to the destination specified from the backend.
   * Set a timestamp instead of the parameter on the store when file form needs to get the latest form status from backend side.
   */
  private requestStoreRefresh() {
    this.store.setDefaultValues({
      [this.parameter().id]: '',
    });
    this.store.set(this.parameter().id, new Date());
  }

  private monitorRefreshingFormStoreWhileVerification() {
    this.formStoreRefreshCancel.next(void 0);
    interval(FileParameterComponent.FORM_STATUS_POLLING_INTERVAL_MS)
      .pipe(
        takeUntil(this.formStoreRefreshCancel),
        takeWhile(() => this.parameter().status === UploadStatus.Verifying),
      )
      .subscribe(() => {
        this.requestStoreRefresh();
      });
  }

  ngOnDestroy(): void {
    this.formStoreRefreshCancel.next(void 0);
  }
}
