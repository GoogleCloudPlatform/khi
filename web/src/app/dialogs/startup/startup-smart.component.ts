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

import { Component, inject, computed } from '@angular/core';
import {
  MatDialog,
  MatDialogRef,
  MatDialogConfig,
} from '@angular/material/dialog';
import { interval, startWith, map } from 'rxjs';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionMetadataDialogComponent } from '../inspection-metadata/inspection-metadata.component';
import { openNewInspectionDialog } from '../new-inspection/new-inspection.component';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { environment } from 'src/environments/environment';
import { VERSION } from 'src/environments/version';
import { toSignal } from '@angular/core/rxjs-interop';
import { BackendAPIUtil } from 'src/app/services/api/backend-api.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import { StartupDialogLayoutComponent } from './components/startup-dialog-layout.component';
import { SidebarLink } from './types/startup-side-menu.types';
import { InspectionListItemViewModel } from './types/inspection-activity.model';
import {
  InspectionMetadataProgressPhase,
  InspectionMetadataProgressElement,
  InspectionMetadataError,
} from 'src/app/common/schema/metadata-types';
import { BackendSyncService } from 'src/app/services/api/backend-sync-interface';

/**
 * Smart component for the Startup Dialog.
 * Handles state management and data fetching.
 */
@Component({
  selector: 'khi-startup-smart',
  imports: [StartupDialogLayoutComponent],
  templateUrl: './startup-smart.component.html',
  styleUrls: ['./startup-smart.component.scss'],
  host: { style: 'display: contents;' },
})
export class StartupDialogSmartComponent {
  private readonly dialog = inject(MatDialog);
  private readonly dialogRef = inject<MatDialogRef<void>>(MatDialogRef);
  private readonly backendAPI = inject<BackendAPI>(BACKEND_API);
  private readonly backendSync = inject<BackendSyncService>(BACKEND_SYNC);
  private readonly loader = inject(InspectionDataLoaderService);
  private readonly progress = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );

  /**
   * The interval to refresh the start time of each tasks written as `xx seconds ago`.
   */
  static readonly UI_TIME_REFRESH_INTERVAL = 1000;

  protected readonly version = VERSION;

  protected readonly links: SidebarLink[] = environment.links;

  protected readonly tasks = this.backendSync.tasks.value;

  protected readonly isLoading = computed(() => this.tasks() === undefined);

  protected readonly isViewerMode = toSignal(
    this.backendAPI.getConfig().pipe(map((config) => config.viewerMode)),
    { initialValue: false },
  );

  private readonly ticker = toSignal(
    interval(StartupDialogSmartComponent.UI_TIME_REFRESH_INTERVAL).pipe(
      startWith(0),
    ),
  );

  protected readonly vmTasks = computed(() => {
    this.ticker(); // register dependency
    const tp = this.tasks();
    if (!tp) return [] as InspectionListItemViewModel[];

    const keys = Object.keys(tp.inspections).sort(
      (a, b) =>
        tp.inspections[a].header.inspectTimeUnixSeconds -
        tp.inspections[b].header.inspectTimeUnixSeconds,
    );
    return keys.map((key) => {
      const taskMetadata = tp.inspections[key];
      return {
        id: key,
        label: taskMetadata.header.inspectionName,
        inspectionTimeLabel: this.durationToTimeSeconds(
          Date.now() - taskMetadata.header.inspectTimeUnixSeconds * 1000,
        ),
        phase: taskMetadata.progress.phase as InspectionMetadataProgressPhase,
        totalProgress: {
          id: key + '-' + taskMetadata.progress.totalProgress.id,
          label: taskMetadata.progress.totalProgress.label,
          message: taskMetadata.progress.totalProgress.message,
          percentage: taskMetadata.progress.totalProgress.percentage * 100,
          percentageLabel: (
            taskMetadata.progress.totalProgress.percentage * 100
          ).toFixed(2),
          indeterminate: false,
        },
        progresses: taskMetadata.progress.progresses.map(
          (p: InspectionMetadataProgressElement) => ({
            id: key + '-' + p.id,
            label: p.label,
            message: p.message,
            percentage: p.percentage * 100,
            percentageLabel: (p.percentage * 100).toFixed(2),
            indeterminate: p.indeterminate,
          }),
        ),
        errors: taskMetadata.error.errorMessages.map(
          (msg: InspectionMetadataError) => ({
            message: msg.message,
            link: msg.link || '',
          }),
        ),
      } as InspectionListItemViewModel;
    });
  });

  protected openNewInspectionDialog() {
    this.openNewInspectionDialogInternal();
  }

  protected openKhiFile() {
    this.loader.uploadFromFile();
    this.dialogRef.close();
  }

  protected cancelTask(id: string) {
    this.backendAPI.cancelInspection(id).subscribe(() => {
      console.log(`task ${id} was cancelled`);
    });
  }

  protected openTaskResult(id: string) {
    this.loader.loadInspectionDataFromBackend(id);
    this.dialogRef.close();
  }

  protected showMetadata(id: string) {
    this.backendAPI.getInspectionMetadata(id).subscribe((metadata) => {
      this.dialog.open(InspectionMetadataDialogComponent, {
        data: metadata,
        maxHeight: 600,
      });
    });
  }

  protected downloadInspectionResult(id: string) {
    BackendAPIUtil.downloadInspectionDataAsFile(
      this.backendAPI,
      id,
      this.progress,
    ).subscribe(() => {
      console.log(`inspection file for task ${id} was downloaded`);
    });
  }

  protected updateInspectionTitle(event: { id: string; changeTo: string }) {
    this.backendAPI
      .patchInspection(event.id, { name: event.changeTo })
      .subscribe(() => {
        console.log(`inspection title for task ${event.id} was updated`);
      });
  }

  private durationToTimeSeconds(duration: number): string {
    const hour = 1000 * 60 * 60;
    const minute = 1000 * 60;
    if (duration >= hour) {
      return `${Math.floor(duration / hour)}h ago`;
    } else if (duration >= minute) {
      return `${Math.floor(duration / minute)}min ago`;
    } else {
      return `${Math.floor(duration / 1000)}s ago`;
    }
  }

  private openNewInspectionDialogInternal() {
    openNewInspectionDialog(this.dialog);
  }
}

/**
 * Opens the Startup Dialog with default configurations.
 * @param dialog MatDialog service instance.
 * @param config Optional dialog configuration to override defaults.
 * @returns MatDialogRef for the opened dialog.
 */
export function openStartupDialog(
  dialog: MatDialog,
  config: Partial<MatDialogConfig> = {},
) {
  return dialog.open(StartupDialogSmartComponent, {
    maxWidth: '100vw',
    minWidth: '900px',
    maxHeight: '600px',
    panelClass: 'startup-modalbox',
    ...config,
  });
}
