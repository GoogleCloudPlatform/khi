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

import { Component, inject } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { combineLatest, interval, map, startWith } from 'rxjs';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionMetadataDialogComponent } from '../inspection-metadata/inspection-metadata.component';
import { openNewInspectionDialog } from '../new-inspection/new-inspection.component';
import { StartupHeaderComponent } from './components/startup-header.component';
import { MatIconModule } from '@angular/material/icon';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatButtonModule } from '@angular/material/button';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import { environment } from 'src/environments/environment';
import { VERSION } from 'src/environments/version';
import { toSignal } from '@angular/core/rxjs-interop';
import { TaskCardListComponent } from './components/task-card-list.component';
import {
  TaskCardItemProgressBarViewModel,
  TaskCardItemViewModel,
} from './components/task-card-item.component';
import { BackendAPIUtil } from 'src/app/services/api/backend-api.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';

/**
 * StartupComponent is a dialog shown just after starting KHI
 */
@Component({
  selector: 'khi-startup',
  templateUrl: './startup.component.html',
  styleUrls: ['./startup.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    MatIconModule,
    MatTooltipModule,
    MatButtonModule,
    StartupHeaderComponent,
    TaskCardListComponent,
  ],
})
export class StartupDialogComponent {
  private readonly dialog = inject(MatDialog);
  private readonly dialogRef = inject<MatDialogRef<void>>(MatDialogRef);
  private readonly backendAPI = inject<BackendAPI>(BACKEND_API);
  private readonly backendConnection =
    inject<BackendConnectionService>(BACKEND_CONNECTION);
  private readonly loader = inject(InspectionDataLoaderService);
  private readonly progress = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );

  /**
   * The interval to refresh the start time of each tasks written as `xx seconds ago`.
   */
  static UI_TIME_REFRESH_INTERVAL = 1000;

  isViewerMode = toSignal(
    this.backendAPI.getConfig().pipe(map((v) => v.viewerMode)),
    { initialValue: false },
  );

  bugReportUrl = environment.bugReportUrl;

  documentUrl = environment.documentUrl;

  tasks = this.backendConnection.tasks();

  version = VERSION;

  vmTasks = toSignal(
    combineLatest([
      // to update task time like `xx seconds ago`
      interval(StartupDialogComponent.UI_TIME_REFRESH_INTERVAL).pipe(
        startWith(0),
      ),
      this.tasks,
    ]).pipe(
      map(([, tp]) => {
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
            phase: taskMetadata.progress.phase,
            totalProgress: {
              id: key + '-' + taskMetadata.progress.totalProgress.id,
              label: taskMetadata.progress.totalProgress.label,
              percentage: taskMetadata.progress.totalProgress.percentage * 100,
              percentageLabel: taskMetadata.progress.totalProgress.message,
              indeterminate: false,
              message: '',
            },
            progresses: taskMetadata.progress.progresses.map(
              (p) =>
                ({
                  id: key + '-' + p.id,
                  label: p.label,
                  message: p.message,
                  percentage: p.percentage * 100,
                  percentageLabel: (p.percentage * 100).toFixed(2),
                  indeterminate: p.indeterminate,
                }) as TaskCardItemProgressBarViewModel,
            ),
            inspectionTimeLabel: this.durationToTimeSeconds(
              Date.now() - taskMetadata.header.inspectTimeUnixSeconds * 1000,
            ),
            errors: taskMetadata.error.errorMessages.map((msg) => ({
              message: msg.message,
              link: msg.link,
            })),
          } as TaskCardItemViewModel;
        });
      }),
    ),
  );

  serverStat = this.tasks.pipe(map((resp) => resp.serverStat));

  openNewInspectionDialog() {
    openNewInspectionDialog(this.dialog);
  }

  openKhiFile() {
    this.loader.uploadFromFile();
    this.dialogRef.close();
  }

  cancelTask(id: string) {
    this.backendAPI.cancelInspection(id).subscribe(() => {
      console.log(`task ${id} was cancelled`);
    });
  }

  openTaskResult(id: string) {
    this.loader.loadInspectionDataFromBackend(id);
    this.dialogRef.close();
  }

  showMetadata(id: string) {
    this.backendAPI.getInspectionMetadata(id).subscribe((metadata) => {
      this.dialog.open(InspectionMetadataDialogComponent, {
        data: metadata,
        maxHeight: 600,
      });
    });
  }

  downloadInspectionResult(id: string) {
    BackendAPIUtil.downloadInspectionDataAsFile(
      this.backendAPI,
      id,
      this.progress,
    ).subscribe(() => {
      console.log(`inspection file for task ${id} was downloaded`);
    });
  }

  updateInspectionTitle(event: { id: string; changeTo: string }) {
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
}
