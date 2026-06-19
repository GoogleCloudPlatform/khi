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

import { Component, model } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CelGuideOverviewComponent } from 'src/app/timeline-toolbar/components/cel-guide-overview.component';
import { CelGuideTimelineCelComponent } from 'src/app/timeline-toolbar/components/cel-guide-timeline-cel.component';
import { CelGuideLogCelComponent } from 'src/app/timeline-toolbar/components/cel-guide-log-cel.component';

/**
 * Tab identifiers for the CEL guide.
 */
export enum CelGuideTab {
  Overview = 'Overview',
  TimelineCel = 'TimelineCel',
  LogCel = 'LogCel',
}

/**
 * Container dialog popup featuring tabs for basic concepts, Timeline CEL, and Log CEL guides.
 */
@Component({
  selector: 'khi-cel-guide-popup',
  templateUrl: './cel-guide-popup.component.html',
  styleUrls: ['./cel-guide-popup.component.scss'],
  imports: [
    CommonModule,
    CelGuideOverviewComponent,
    CelGuideTimelineCelComponent,
    CelGuideLogCelComponent,
  ],
})
export class CelGuidePopupComponent {
  protected readonly CelGuideTab = CelGuideTab;

  /**
   * Controls the active tab state via two-way binding model.
   */
  readonly activeTab = model<CelGuideTab>(CelGuideTab.Overview);
}
