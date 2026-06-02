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

import { Injectable, inject } from '@angular/core';
import { WindowConnectorService } from '../window-connector.service';
import {
  DIFF_PAGE_OPEN,
  UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
  UpdateSelectedResourceMessage,
} from 'src/app/common/schema/inter-window-messages';
import { SelectionManagerV2 } from '../../selection-manager-v2.service';

/**
 * DiffPageDataSourceServer sends data needed to show the diff page in the other tab.
 */
@Injectable()
export class DiffPageDataSourceServer {
  private connector = inject(WindowConnectorService);
  private selectionManager = inject(SelectionManagerV2);

  public activate() {
    // Send the current selected revision and timeline to newly activated diff page
    this.connector.receiver(DIFF_PAGE_OPEN).subscribe((message) => {
      const revision = this.selectionManager.selectedRevision();
      const timeline = this.selectionManager.selectedTimeline();

      if (timeline && revision) {
        this.connector.unicast<UpdateSelectedResourceMessage>(
          UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
          {
            timelinePath: timeline.path,
            previousContent: revision.prev ? revision.prev.bodyYAML : '',
            currentContent: revision.bodyYAML,
            logIndex: revision.logIndex,
          },
          message.sourceFrameId!,
        );
      }
    });
  }
}
