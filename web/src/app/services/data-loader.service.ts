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
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { ProgressReporter } from 'src/app/services/progress/progress-interface';
import { ImportInspectionClientService } from 'src/app/services/api/import-inspection-client.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';

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
  private readonly extension = inject<ExtensionStore>(EXTENSION_STORE);
  private readonly importClient = inject(ImportInspectionClientService);
  private readonly workbenchClient = inject(WorkbenchClientService);

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

      this.progress.dismiss();
    } catch (e) {
      this.progress.dismiss();
      console.error(e);
      alert(
        `Failed to import inspection file: ${e instanceof Error ? e.message : 'Unknown error'}. Please verify that the server is running and the file is a valid .khi file.`,
      );
    }
  }

  public async loadInspectionDataFromBackend(inspectionID: string) {
    const sessionMatch =
      typeof window !== 'undefined'
        ? window.location.pathname.match(/\/session\/([^/]+)/)
        : null;
    const sessionId = sessionMatch ? sessionMatch[1] : '0';

    this.progress.show();

    let localPercent = 0;
    let localMessage = 'Downloading inspection data...';
    let serverPercent = 0;
    let serverMessage = 'Initializing server workbench...';

    const updateCombinedProgress = () => {
      const combinedPercent = Math.min(
        100,
        Math.round(localPercent * 0.5 + serverPercent * 0.5),
      );
      this.progress.updateProgress({
        message: '',
        percent: combinedPercent,
        mode: 'determinate',
        tasks: [
          {
            id: 'client',
            label: 'Client',
            message: localMessage,
            percent: Math.round(localPercent),
            mode: 'determinate',
          },
          {
            id: 'server',
            label: 'Server',
            message: serverMessage,
            percent: Math.round(serverPercent),
            mode: 'determinate',
          },
        ],
      });
    };

    updateCombinedProgress();

    // Task 1: Server Workbench initialization
    const serverTask = this.workbenchClient.openWorkbench(
      sessionId,
      inspectionID,
      (msg, pct) => {
        serverMessage = msg;
        serverPercent = pct;
        updateCombinedProgress();
      },
    );

    // Task 2: Local download and parsing
    const localTask = (async (): Promise<{
      parsedData: InspectionData;
      rawInspectionData: ArrayBuffer;
    }> => {
      const data = await lastValueFrom(
        this.backendService.getInspectionData(inspectionID, (allSize, done) => {
          const downloadRatio = allSize > 0 ? done / allSize : 0;
          localPercent = downloadRatio * 50;
          localMessage = `Downloading data (${ProgressUtil.formatPogressMessageByBytes(
            done,
            allSize,
          )})`;
          updateCombinedProgress();
        }),
      );

      localPercent = 50;
      localMessage = 'Parsing inspection data...';
      updateCombinedProgress();

      const rawInspectionData = await data.content.arrayBuffer();
      const parser = new KHIFileParser({ 6: V6_BLUEPRINT });
      const progressReporter: ProgressReporter = {
        reportProgress: (percent?: number) => {
          if (typeof percent === 'number') {
            localPercent = 50 + (percent / 100) * 50;
          }
          updateCombinedProgress();
        },
        reportMessage: (message: string) => {
          localMessage = message;
          updateCombinedProgress();
        },
        complete: () => {
          localPercent = 100;
          updateCombinedProgress();
        },
      };

      const parsedData = await parser.parse(
        rawInspectionData,
        progressReporter,
      );
      return { parsedData, rawInspectionData };
    })();

    try {
      const [{ parsedData, rawInspectionData }] = await Promise.all([
        localTask,
        serverTask,
      ]);
      this.inspectionDataStore.setNewInspectionData(parsedData);
      this.extension.notifyLifecycleOnInspectionDataOpen(
        parsedData,
        rawInspectionData,
      );
    } catch (e) {
      console.error(e);
      alert(
        `Failed to load the inspection data or initialize workbench. The given data was invalid or too big for this environment. \nPlease consider limiting the inspection duration shorter.`,
      );
    } finally {
      this.progress.dismiss();
    }
  }
}
