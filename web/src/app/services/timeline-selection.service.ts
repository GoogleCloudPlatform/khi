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

import { Injectable } from '@angular/core';
import { BehaviorSubject, filter, firstValueFrom, map } from 'rxjs';
import { InspectionDataStoreService } from './inspection-data-store.service';
import { SelectionManagerService } from './selection-manager.service';

@Injectable({ providedIn: 'root' })
export class TimelineSelectionService {
  private logs = this._inspectionDataStore.allLogs;

  private $currentTime: BehaviorSubject<number> = new BehaviorSubject(0);

  constructor(
    private _inspectionDataStore: InspectionDataStoreService,
    private _logSelectionManager: SelectionManagerService,
  ) {
    this._logSelectionManager.selectedLog
      .pipe(
        filter((log) => log !== null),
        map((log) => log!.time),
      )
      .subscribe(this.$currentTime);
  }

  public seek(diff: number) {
    if (diff > 0) {
      this.seekToAfter(this.$currentTime.value + diff);
    } else {
      this.seekToBefore(this.$currentTime.value + diff);
    }
  }

  public async seekToBefore(time: number) {
    const logs = await firstValueFrom(this.logs);
    if (logs.length == 0) return;
    if (logs[logs.length - 1].time < time) {
      this._logSelectionManager.changeSelectionByLog(logs.length - 1);
    } else {
      let left = 0;
      let right = logs.length - 1;
      while (right - left > 1) {
        const mid = Math.floor((left + right) / 2);
        if (logs[mid].time <= time) {
          left = mid;
        } else {
          right = mid;
        }
      }
      this._logSelectionManager.changeSelectionByLog(left);
    }
  }

  public async seekToAfter(time: number) {
    const logs = await firstValueFrom(this.logs);
    if (logs.length == 0) return;
    if (logs[0].time > time) {
      this._logSelectionManager.changeSelectionByLog(0);
    } else {
      let left = 0;
      let right = logs.length - 1;
      while (right - left > 1) {
        const mid = Math.floor((left + right) / 2);
        if (logs[mid].time < time) {
          left = mid;
        } else {
          right = mid;
        }
      }
      this._logSelectionManager.changeSelectionByLog(right);
    }
  }
}
