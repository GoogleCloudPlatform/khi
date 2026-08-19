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
import { lastValueFrom } from 'rxjs';
import { BACKEND_API, BackendAPI } from './api/backend-api-interface';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from './progress/progress-interface';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from '../extensions/extension-common/extension-store';
import { ProgressUtil } from './progress/progress-util';
import { KHIFileParser } from 'src/app/parser/core/file-parser';
import { V6_BLUEPRINT } from 'src/app/parser/v6/blueprint';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { ProgressReporter } from 'src/app/services/progress/progress-interface';
import { ImportInspectionClientService } from 'src/app/services/api/import-inspection-client.service';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendSyncService } from 'src/app/services/api/backend-sync-interface';

/**
 * Service for importing, downloading, and parsing inspection data.
 */
@Injectable()
export class InspectionDataLoaderService {
  private readonly progress = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly backendService = inject<BackendAPI>(BACKEND_API);
  private readonly backendSync = inject<BackendSyncService>(BACKEND_SYNC, {
    optional: true,
  });
  private readonly extension = inject<ExtensionStore>(EXTENSION_STORE);
  private readonly importClient = inject(ImportInspectionClientService);

  /**
   * Opens file selector dialog to pick a local .khi file and uploads it to the backend server.
   */
  public uploadFromFile() {
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.accept = '.khi';
    fileInput.style.display = 'none';
    document.body.appendChild(fileInput);
    fileInput.oninput = () => {
      const file = fileInput.files?.[0];
      if (file) {
        this.importInspectionFile(file);
      }
      fileInput.remove();
    };
    fileInput.click();
  }

  /**
   * Imports a .khi file in chunks to the backend server.
   *
   * @param file The .khi inspection file.
   */
  public async importInspectionFile(file: File) {
    this.progress.show();
    this.progress.updateProgress({
      message: `Uploading inspection file (${ProgressUtil.formatPogressMessageByBytes(0, file.size)})...`,
      percent: 0,
      mode: 'determinate',
    });

    try {
      await this.importClient.importFile(file, {
        onProgress: (uploaded, total) => {
          this.progress.updateProgress({
            message: `Uploading inspection file (${ProgressUtil.formatPogressMessageByBytes(uploaded, total)})...`,
            percent: (uploaded / total) * 100,
            mode: 'determinate',
          });
        },
      });

      this.backendSync?.tasks.reload();
      this.progress.dismiss();
    } catch (e) {
      this.progress.dismiss();
      console.error(e);
      alert(
        `Failed to import inspection file: ${e instanceof Error ? e.message : 'Unknown error'}. Please verify that the server is running and the file is a valid .khi file.`,
      );
    }
  }

  /**
   * Downloads and loads an inspection dataset from the backend server.
   *
   * @param inspectionID Unique ID of the inspection.
   */
  public async loadInspectionDataFromBackend(inspectionID: string) {
    this.progress.show();
    this.progress.updateProgress({
      message: 'Downloading inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    try {
      const data = await lastValueFrom(
        this.backendService.getInspectionData(inspectionID, (allSize, done) => {
          this.progress.updateProgress({
            message: `Downloading inspection data...(${ProgressUtil.formatPogressMessageByBytes(
              done,
              allSize,
            )})`,
            percent: (done / allSize) * 100,
            mode: 'determinate',
          });
        }),
      );
      this.progress.dismiss();
      await this.parseAndStoreInspectionData(await data.content.arrayBuffer());
    } catch (e) {
      console.error(e);
      alert(
        `Failed to load the inspection data. Please try query with shorter duration.`,
      );
    }
  }

  /**
   * Parses raw binary inspection data into the InspectionDataStore.
   *
   * @param rawInspectionData Binary inspection data.
   */
  private async parseAndStoreInspectionData(rawInspectionData: ArrayBuffer) {
    this.progress.show();
    this.progress.updateProgress({
      message: 'Parsing inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    try {
      const parser = new KHIFileParser({ 6: V6_BLUEPRINT });
      const progressReporter: ProgressReporter = {
        reportProgress: (percent?: number) => {
          this.progress.updateProgress({
            percent: percent ?? 0,
            message: 'Parsing inspection data...',
            mode: typeof percent === 'number' ? 'determinate' : 'indeterminate',
          });
        },
        reportMessage: (message: string) => {
          this.progress.updateProgress({
            percent: 0,
            message,
            mode: 'indeterminate',
          });
        },
        complete: () => {},
      };
      const parsedData = await parser.parse(
        rawInspectionData,
        progressReporter,
      );
      this.inspectionDataStore.setNewInspectionData(parsedData);
      this.extension.notifyLifecycleOnInspectionDataOpen(
        parsedData,
        rawInspectionData,
      );
    } catch (e) {
      console.error(e);
      alert(
        `Failed to parse the inspection data. The given data was invalid or too big for this environment. \nPlease consider limiting the inspection duration shorter.`,
      );
    }
    this.progress.dismiss();
  }
}
