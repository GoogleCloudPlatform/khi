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

import { Component, OnDestroy, effect, inject, signal } from '@angular/core';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { IndexProgressLayoutComponent } from './components/index-progress-layout.component';

/**
 * Smart component managing visibility and data for the bottom-right search index progress overlay.
 */
@Component({
  selector: 'khi-index-progress-smart',
  imports: [IndexProgressLayoutComponent],
  templateUrl: './index-progress-smart.component.html',
  styleUrls: ['./index-progress-smart.component.scss'],
  host: { style: 'display: contents;' },
})
export class IndexProgressSmartComponent implements OnDestroy {
  private readonly workbenchClient = inject(WorkbenchClientService);

  private readonly isVisibleSignal = signal<boolean>(false);
  private dismissTimeout: ReturnType<typeof setTimeout> | null = null;

  /**
   * Whether the progress overlay is currently visible.
   */
  public readonly visible = this.isVisibleSignal.asReadonly();

  /**
   * Indexing progress percentage.
   */
  public readonly percent = this.workbenchClient.indexProgressPercentage;

  /**
   * Current indexing status message.
   */
  public readonly message = this.workbenchClient.indexMessage;

  /**
   * Whether the index is complete and ready.
   */
  public readonly isReady = this.workbenchClient.isIndexReady;

  constructor() {
    effect(
      () => {
        const isBuilding = this.workbenchClient.isIndexBuilding();
        const isReady = this.workbenchClient.isIndexReady();

        if (isBuilding) {
          if (this.dismissTimeout) {
            clearTimeout(this.dismissTimeout);
            this.dismissTimeout = null;
          }
          this.isVisibleSignal.set(true);
        } else if (isReady && this.isVisibleSignal()) {
          if (!this.dismissTimeout) {
            this.dismissTimeout = setTimeout(() => {
              this.isVisibleSignal.set(false);
              this.dismissTimeout = null;
            }, 1500);
          }
        } else {
          if (!isReady) {
            this.isVisibleSignal.set(false);
          }
        }
      },
      { allowSignalWrites: true },
    );
  }

  /**
   * Cleans up pending timers on destroy.
   */
  public ngOnDestroy(): void {
    if (this.dismissTimeout) {
      clearTimeout(this.dismissTimeout);
      this.dismissTimeout = null;
    }
  }
}
