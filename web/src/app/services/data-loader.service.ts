/**
 * Copyright 2024 Google LLC
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
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { ProgressReporter } from 'src/app/services/progress/progress-interface';

@Injectable()
export class InspectionDataLoaderService {
  private readonly progress = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );
  private readonly inspectionDataStoreV2 = inject(InspectionDataStoreV2);
  private readonly backendService = inject<BackendAPI>(BACKEND_API);
  private readonly extension = inject<ExtensionStore>(EXTENSION_STORE);

  /**
   * Open a dialog to open local file and accept that JSON as the inspection data.
   */
  public uploadFromFile() {
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.style.display = 'none';
    document.body.appendChild(fileInput);
    fileInput.oninput = () => {
      const fileReader = new FileReader();
      fileReader.onload = () => {
        this.loadInspectionDataDirect(fileReader.result as ArrayBuffer);
        location.hash = '';
        fileInput.remove();
      };
      fileReader.readAsArrayBuffer(fileInput.files![0]);
    };
    fileInput.click();
  }

  public async loadInspectionDataDirect(rawInspectionData: ArrayBuffer) {
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
      this.inspectionDataStoreV2.setNewInspectionData(parsedData);
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
      this.loadInspectionDataDirect(await data.content.arrayBuffer());
    } catch (e) {
      console.error(e);
      // Since the file size could be large, there could be a several reasons to fail including browser limtations.
      // Smaller file size should always be an option.
      alert(
        `Failed to load the inspection data. Please try query with shorter duration.`,
      );
    }
  }
}
