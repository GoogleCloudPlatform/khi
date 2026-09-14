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

import { ChangeDetectionStrategy, Component } from '@angular/core';
import { DagNodeRunPhase } from 'src/app/shared/components/dag-viewer/dag-viewer.model';

/**
 * Single entry of the run phase legend.
 */
interface DagRunPhaseLegendItem {
  /**
   * Run phase this entry explains.
   */
  readonly runPhase: DagNodeRunPhase;

  /**
   * Human-readable label shown next to the color swatch.
   */
  readonly label: string;
}

/**
 * Legend explaining how each task run phase is decorated on the DAG canvas.
 */
@Component({
  selector: 'khi-dag-run-phase-legend',
  templateUrl: './dag-run-phase-legend.component.html',
  styleUrls: ['./dag-run-phase-legend.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DagRunPhaseLegendComponent {
  /**
   * Legend entries rendered in execution order.
   */
  protected readonly items: readonly DagRunPhaseLegendItem[] = [
    { runPhase: DagNodeRunPhase.WAITING, label: 'Waiting' },
    { runPhase: DagNodeRunPhase.RUNNING, label: 'Running' },
    { runPhase: DagNodeRunPhase.DONE, label: 'Done' },
    { runPhase: DagNodeRunPhase.ERROR, label: 'Error' },
  ];
}
