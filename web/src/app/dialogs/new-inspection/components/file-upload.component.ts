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
  signal,
  ViewChild,
} from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import {
  FILE_UPLOADER,
  UploadToken,
  UploadStatus,
} from './service/file-uploader';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

@Component({
  selector: 'khi-new-inspection-file-upload',
  templateUrl: './file-upload.component.html',
  styleUrls: ['./file-upload.component.sass'],
  imports: [
    CommonModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    MatIconModule,
    MatButtonModule,
    MatSnackBarModule,
    MatProgressSpinnerModule,
  ],
})
export class FileUploadComponent {
  /**
   * The label of this upload form field.
   */
  label = input.required<string>();

  /**
   * The token provided from backend to request file.
   */
  uploadToken = input.required<UploadToken>();

  /**
   * The description of this upload form field.
   */
  description = input('');

  /**
   * The error message returned from the backend about this field.
   */
  errorMessage = input('');

  /**
   * The status of upload for this file form.
   */
  uploadStatus = input(UploadStatus.Waiting);

  /**
   * The state if currently selected file is uploaded or not.
   * This parameter is a status only for the currently selected file, `uploadStatus` is a status for this field.
   * This parameter will be false even if `uploadStatus` is completed when user successfully uploaded their file on the server but opened a new file on the form.
   */
  isSelectedFileUploaded = signal(true);

  /**
   * The status if a file is dragged over the file dropping area.
   */
  fileDraggingOverArea = signal(false);

  /**
   * The ratio of file size completed upload.
   */
  uploadRatio = signal(0);

  /**
   * The filename uploaded or will be uploaded on this field.
   * This state directly hold by FileUploadComponent and not used except for users to know which they uploaded.
   */
  filename = signal('');

  @ViewChild('fileInput')
  fileInput!: ElementRef<HTMLInputElement>;

  selectedFile: File | null = null;

  private snackBar = inject(MatSnackBar);

  private uploader = inject(FILE_UPLOADER);

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
    this.uploader
      .upload(this.uploadToken(), this.selectedFile)
      .subscribe((status) => {
        this.uploadRatio.set(status.completeRatio);
        if (status.done) {
          this.isSelectedFileUploaded.set(true);
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
}
