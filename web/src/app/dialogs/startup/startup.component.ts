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

import { Component, Inject } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import {
  ReplaySubject,
  Subject,
  combineLatest,
  interval,
  map,
  of,
  shareReplay,
  startWith,
  switchMap,
  withLatestFrom,
} from 'rxjs';
import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';
import { BackendAPIUtil } from 'src/app/services/api/backend-api.service';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionMetadataDialogComponent } from '../inspection-metadata/inspection-metadata.component';
import { openNewInspectionDialog } from '../new-inspection/new-inspection.component';
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
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import { VERSION } from 'src/environments/version';

export type ProgressBarViewModel = {
  id: string;
  label: string;
  message: string;
  percentage: number;
  percentageLabel: string;
};

export type ErrorViewModel = {
  message: string;
  link: string;
};

export type TaskListViewModel = {
  id: string;
  inspectionTimeLabel: string;
  label: string;
  iconPath: string;
  phase: InspectionMetadataProgressPhase;
  totalProgress: ProgressBarViewModel;
  progresses: ProgressBarViewModel[];
  downloading: Subject<boolean>;
  errors: ErrorViewModel[];
};

/**
 * StartupComponent is a dialog shown just after starting KHI
 */
@Component({
  selector: 'khi-startup',
  templateUrl: './startup.component.html',
  styleUrls: ['./startup.component.sass'],
  standalone: true,
  imports: [CommonModule, MatIconModule, MatTooltipModule, MatButtonModule],
})
export class StartupDialogComponent {
  /**
   * The interval to refresh the start time of each tasks written as `xx seconds ago`.
   */
  static UI_TIME_REFRESH_INTERVAL = 1000;

  isViewerMode = this.backendAPI.getConfig().pipe(map((v) => v.viewerMode));

  bugReportUrl = environment.bugReportUrl;

  documentUrl = environment.documentUrl;

  tasks = this.backendConnection.tasks();

  version = VERSION;

  taskListViewModel = combineLatest([
    interval(StartupDialogComponent.UI_TIME_REFRESH_INTERVAL).pipe(
      startWith(0),
    ),
    this.tasks,
  ]).pipe(
    map(([, tp]) => {
      const keys = Object.keys(tp.tasks).sort(
        (a, b) =>
          tp.tasks[a].header.inspectTimeUnixSeconds -
          tp.tasks[b].header.inspectTimeUnixSeconds,
      );
      return keys.map((key) => {
        const taskMetadata = tp.tasks[key];
        return {
          id: key,
          label: taskMetadata.header.inspectionType,
          iconPath: taskMetadata.header.inspectionTypeIconPath,
          phase: taskMetadata.progress.phase,
          totalProgress: {
            id: key + '-' + taskMetadata.progress.totalProgress.id,
            label: taskMetadata.progress.totalProgress.label,
            percentage: taskMetadata.progress.totalProgress.percentage * 100,
            percentageLabel: taskMetadata.progress.totalProgress.message,
          },
          progresses: taskMetadata.progress.progresses.map(
            (p) =>
              ({
                id: key + '-' + p.id,
                label: p.label,
                message: p.message,
                percentage: p.percentage * 100,
                percentageLabel: (p.percentage * 100).toFixed(2),
              }) as ProgressBarViewModel,
          ),
          inspectionTimeLabel: this.durationToTimeSeconds(
            Date.now() - taskMetadata.header.inspectTimeUnixSeconds * 1000,
          ),
          downloading: new ReplaySubject<boolean>(1).pipe(startWith(false)),
          errors: taskMetadata.error.errorMessages.map((msg) => ({
            message: msg.message,
            link: msg.link,
          })),
        } as TaskListViewModel;
      });
    }),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  serverStat = this.tasks.pipe(map((resp) => resp.serverStat));

  constructor(
    private readonly dialog: MatDialog,
    private readonly dialogRef: MatDialogRef<void>,
    @Inject(BACKEND_API) private readonly backendAPI: BackendAPI,
    @Inject(BACKEND_CONNECTION)
    private readonly backendConnection: BackendConnectionService,
    private readonly loader: InspectionDataLoaderService,
    @Inject(PROGRESS_DIALOG_STATUS_UPDATOR)
    private readonly progress: ProgressDialogStatusUpdator,
  ) {}

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

  downloadInspectionData(id: string) {
    of([null])
      .pipe(
        withLatestFrom(this.taskListViewModel),
        switchMap(([, vm]) => {
          vm.filter((m) => m.id == id).forEach((m) => m.downloading.next(true));
          return BackendAPIUtil.downloadInspectionDataAsFile(
            this.backendAPI,
            id,
            this.progress,
          );
        }),
        withLatestFrom(this.taskListViewModel),
      )
      .subscribe(([, vm]) => {
        vm.filter((m) => m.id == id).forEach((m) => m.downloading.next(false));
      });
  }

  public progressCollectionTrack(
    index: number,
    progress: ProgressBarViewModel,
  ): string {
    return progress.id;
  }

  public taskCollectionTrack(index: number, task: TaskListViewModel): string {
    return task.id;
  }
}
