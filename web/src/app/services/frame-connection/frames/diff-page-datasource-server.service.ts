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
import { WindowConnectorService } from '../window-connector.service';
import {
  DIFF_PAGE_OPEN,
  UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
  UpdateSelectedResourceMessage,
} from 'src/app/common/schema/inter-window-messages';
import { SelectionManagerService } from '../../selection-manager.service';
import { withLatestFrom } from 'rxjs';

/**
 * DiffPageDataSourceServer sends data needed to show the diff page in the other tab.
 */
@Injectable()
export class DiffPageDataSourceServer {
  constructor(
    private connector: WindowConnectorService,
    private selectionManager: SelectionManagerService,
  ) {}

  public activate() {
    // Send the current selected revision and timeline to newly activated diff page
    this.connector
      .receiver(DIFF_PAGE_OPEN)
      .pipe(
        withLatestFrom(
          this.selectionManager.selectedRevision,
          this.selectionManager.selectedTimeline,
        ),
      )
      .subscribe(([message, revision, timeline]) => {
        if (timeline && revision) {
          this.connector.unicast<UpdateSelectedResourceMessage>(
            UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
            {
              timeline,
              logIndex: revision.logIndex,
            },
            message.sourceFrameId!,
          );
        }
      });
  }
}
