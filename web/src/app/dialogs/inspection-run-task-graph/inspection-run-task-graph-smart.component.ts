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

import {
  ChangeDetectionStrategy,
  Component,
  OnDestroy,
  OnInit,
  computed,
  inject,
  signal,
} from '@angular/core';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogConfig,
} from '@angular/material/dialog';
import { InspectionRunTaskGraphLayoutComponent } from 'src/app/dialogs/inspection-run-task-graph/components/inspection-run-task-graph-layout.component';
import {
  computeRunElapsedMs,
  convertToDagViewerEdges,
  convertToDagViewerNodes,
  countFinishedTasks,
} from 'src/app/dialogs/inspection-run-task-graph/inspection-run-task-graph.converter';
import {
  InspectionRunTaskGraphDialogData,
  InspectionRunTaskGraphViewModel,
} from 'src/app/dialogs/inspection-run-task-graph/types/inspection-run-task-graph.viewmodel';
import { InspectionRunTaskGraphSnapshot } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import {
  delayWithSignal,
  isRetryableError,
} from 'src/app/services/api/retry-util';

/**
 * Delay before reconnecting to the observation stream.
 */
const RECONNECT_DELAY_MS = 1000;

/**
 * Number of consecutive stream failures tolerated before the dialog gives up observing.
 */
const MAX_CONSECUTIVE_STREAM_ERRORS = 5;

/**
 * Smart container observing the task graph progress of a single inspection run.
 */
@Component({
  selector: 'khi-inspection-run-task-graph-smart',
  templateUrl: './inspection-run-task-graph-smart.component.html',
  styleUrls: ['./inspection-run-task-graph-smart.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [InspectionRunTaskGraphLayoutComponent],
})
export class InspectionRunTaskGraphSmartComponent implements OnInit, OnDestroy {
  private readonly connectClient = inject(ConnectClientService);
  private readonly dialogData =
    inject<InspectionRunTaskGraphDialogData>(MAT_DIALOG_DATA);
  private readonly abortController = new AbortController();

  private readonly snapshot = signal<InspectionRunTaskGraphSnapshot | null>(
    null,
  );

  private readonly watchErrorMessage = signal<string>('');

  /**
   * Rendering state handed to the layout component.
   */
  protected readonly viewModel = computed<InspectionRunTaskGraphViewModel>(
    () => {
      const snapshot = this.snapshot();
      if (!snapshot) {
        return {
          inspectionName: this.dialogData.inspectionName,
          nodes: [],
          edges: [],
          finishedTaskCount: 0,
          totalTaskCount: 0,
          elapsedMs: 0,
          isRunFinished: false,
          watchErrorMessage: this.watchErrorMessage(),
        };
      }
      const statusByTaskImplementationId = new Map(
        snapshot.nodeStatuses.map((status) => [
          status.taskImplementationId,
          status,
        ]),
      );
      return {
        inspectionName: this.dialogData.inspectionName,
        nodes: convertToDagViewerNodes(
          snapshot.dag?.nodes ?? [],
          statusByTaskImplementationId,
          snapshot.snapshotTimeUnixNano,
        ),
        edges: convertToDagViewerEdges(snapshot.dag?.edges ?? []),
        finishedTaskCount: countFinishedTasks(snapshot.nodeStatuses),
        totalTaskCount: snapshot.nodeStatuses.length,
        elapsedMs: computeRunElapsedMs(snapshot),
        isRunFinished: snapshot.isRunFinished,
        watchErrorMessage: this.watchErrorMessage(),
      };
    },
  );

  ngOnInit(): void {
    void this.watchRunTaskGraph();
  }

  ngOnDestroy(): void {
    this.abortController.abort();
  }

  /**
   * Observes the run until it finishes, reconnecting whenever the server closes the stream.
   *
   * The server closes the stream periodically to bound its lifetime, so a clean end of stream is
   * a normal reconnect trigger rather than a completion signal. Reconnects are always delayed so
   * a stream that closes immediately cannot turn into a tight RPC loop.
   */
  private async watchRunTaskGraph(): Promise<void> {
    const signal = this.abortController.signal;
    let consecutiveErrors = 0;
    while (!signal.aborted) {
      try {
        for await (const res of this.connectClient.inspectionTaskGraphClient.watchInspectionRunTaskGraph(
          { inspectionId: this.dialogData.inspectionId },
          { signal },
        )) {
          if (!res.snapshot) {
            continue;
          }
          consecutiveErrors = 0;
          this.snapshot.set(res.snapshot);
          this.watchErrorMessage.set('');
        }
      } catch (err) {
        if (signal.aborted) {
          return;
        }
        consecutiveErrors++;
        this.watchErrorMessage.set(
          err instanceof Error
            ? err.message
            : 'Failed to observe the inspection run.',
        );
        if (
          consecutiveErrors > MAX_CONSECUTIVE_STREAM_ERRORS ||
          !isRetryableError(err)
        ) {
          return;
        }
      }
      if (this.snapshot()?.isRunFinished) {
        return;
      }
      await delayWithSignal(RECONNECT_DELAY_MS, signal);
    }
  }
}

/**
 * Opens the inspection run task graph dialog for the given inspection.
 *
 * @param dialog MatDialog service instance.
 * @param data Identifier and display name of the inspection to observe.
 * @param config Optional dialog configuration to override defaults.
 * @returns MatDialogRef for the opened dialog.
 */
export function openInspectionRunTaskGraphDialog(
  dialog: MatDialog,
  data: InspectionRunTaskGraphDialogData,
  config: Partial<MatDialogConfig> = {},
) {
  return dialog.open(InspectionRunTaskGraphSmartComponent, {
    data,
    width: '90vw',
    maxWidth: '1400px',
    height: '80vh',
    ...config,
  });
}
