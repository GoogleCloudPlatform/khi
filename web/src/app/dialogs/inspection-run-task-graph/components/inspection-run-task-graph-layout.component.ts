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
  computed,
  input,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { InspectionRunTaskGraphViewModel } from 'src/app/dialogs/inspection-run-task-graph/types/inspection-run-task-graph.viewmodel';
import { DagCanvasComponent } from 'src/app/shared/components/dag-viewer/dag-canvas.component';
import { DagRunPhaseLegendComponent } from 'src/app/shared/components/dag-viewer/dag-run-phase-legend.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { formatDurationMs } from 'src/app/utils/time-format-util';

/**
 * Renders the task graph of a single inspection run together with its progress summary.
 */
@Component({
  selector: 'khi-inspection-run-task-graph-layout',
  templateUrl: './inspection-run-task-graph-layout.component.html',
  styleUrls: ['./inspection-run-task-graph-layout.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIconModule,
    KHIIconRegistrationModule,
    DagCanvasComponent,
    DagRunPhaseLegendComponent,
  ],
})
export class InspectionRunTaskGraphLayoutComponent {
  /**
   * Current rendering state of the observed inspection run.
   */
  readonly viewModel = input.required<InspectionRunTaskGraphViewModel>();

  /**
   * Ratio of finished tasks expressed as a percentage between 0 and 100.
   */
  protected readonly progressPercentage = computed<number>(() => {
    const { finishedTaskCount, totalTaskCount } = this.viewModel();
    if (totalTaskCount === 0) {
      return 0;
    }
    return (finishedTaskCount / totalTaskCount) * 100;
  });

  /**
   * Finished task count against the total task count.
   */
  protected readonly progressLabel = computed<string>(() => {
    const { finishedTaskCount, totalTaskCount } = this.viewModel();
    return `${finishedTaskCount} / ${totalTaskCount} tasks`;
  });

  /**
   * Wall clock time spent by the run, formatted for display.
   */
  protected readonly elapsedLabel = computed<string>(() =>
    formatDurationMs(this.viewModel().elapsedMs),
  );

  /**
   * Short status word describing whether the run is still in progress.
   */
  protected readonly runStateLabel = computed<string>(() =>
    this.viewModel().isRunFinished ? 'Finished' : 'Running',
  );
}
