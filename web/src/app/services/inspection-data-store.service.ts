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

import { Injectable, signal, inject } from '@angular/core';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { TimelineView } from 'src/app/store/domain/timeline-view';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';

/**
 * Service to store and manage the active InspectionData domain model.
 * Provides access to the parsed data and derived timeline views.
 */
@Injectable({ providedIn: 'root' })
export class InspectionDataStore {
  private readonly _inspectionData = signal<InspectionData | null>(null);
  private readonly _timelineView = signal<TimelineView | null>(null);
  private readonly workbenchClient = inject(WorkbenchClientService);

  /**
   * Holds the active inspection data. Emits null if no data has been loaded.
   */
  public readonly inspectionData = this._inspectionData.asReadonly();

  /**
   * Exposes the primary TimelineView parsed and processed from the current timeline store.
   * Automatically updates when the underlying inspection data is changed.
   */
  public readonly timelineView = this._timelineView.asReadonly();

  /**
   * Updates the store with the newly provided inspection data.
   *
   * @param data The new inspection data instance to set.
   */
  public setNewInspectionData(data: InspectionData): void {
    this.updateStoreData(data);
  }

  private updateStoreData(data: InspectionData | null): void {
    this._inspectionData.set(data);
    if (!data) {
      this._timelineView.set(null);
      return;
    }

    const view = new TimelineView(data.timelineStore, this.workbenchClient);
    this._timelineView.set(view);
  }
}
