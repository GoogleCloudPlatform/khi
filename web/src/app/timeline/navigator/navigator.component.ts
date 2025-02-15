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

import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { map, shareReplay } from 'rxjs';
import { EXTENSION_STORE } from 'src/app/extensions/extension-common/extension-store';
import { DisplayableTimelineNavigatorExtension } from 'src/app/extensions/extension-common/extension-types/timeline-navigator';
import { SelectionManagerService } from 'src/app/services/selection-manager.service';
import { ResourceTimeline } from 'src/app/store/timeline';

interface NavigatorLayer {
  label: string;
  icon: string;
  isLast: boolean;
  extensions: DisplayableTimelineNavigatorExtension[];
}

/**
 * NavigatorComponent is a control shown bottom left of timeline chart.
 * Contains information for currently selected timeline.
 */
@Component({
  templateUrl: './navigator.component.html',
  styleUrl: './navigator.component.sass',
  selector: 'khi-timeline-navigator',
  imports: [CommonModule, MatIconModule],
})
export class NavigatorComponent {
  private readonly extensionStore = inject(EXTENSION_STORE);
  private readonly selectionManager = inject(SelectionManagerService);
  selectedTimeline = this.selectionManager.selectedTimeline;

  /**
   * Array of timelines in the path between selected timeline and its root.
   */
  timelinesInHierarchyPath = this.selectedTimeline.pipe(
    map((tl) => {
      const layers: ResourceTimeline[] = [];
      while (tl) {
        layers.push(tl);
        tl = tl.parent;
      }
      return layers.reverse();
    }),
    shareReplay(1),
  );

  navigatorLayers = this.timelinesInHierarchyPath.pipe(
    map((tls) =>
      tls.map(
        (tl, index) =>
          ({
            label: tl.name,
            icon: '',
            isLast: index == tls.length - 1,
            extensions:
              this.extensionStore.getVisibleTimelineNavigatorExtensions(tl),
          }) as NavigatorLayer,
      ),
    ),
  );
}
