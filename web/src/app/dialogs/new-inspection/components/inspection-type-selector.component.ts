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

import { Component, input, output } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { BreaklinePipe } from 'src/app/common/breakline.pipe';
import { InspectionType } from 'src/app/common/schema/api-types';

/**
 * Dumb component for selecting an inspection type in Step 1 of the new inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-type-selector',
  templateUrl: './inspection-type-selector.component.html',
  styleUrls: ['./inspection-type-selector.component.scss'],
  imports: [MatCardModule, MatProgressBarModule, BreaklinePipe],
})
export class InspectionTypeSelectorComponent {
  /**
   * The list of inspection types available for selection, or null if still loading.
   */
  readonly inspectionTypes = input<readonly InspectionType[] | null>(null);

  /**
   * Emits the selected inspection type when a user clicks on an inspection type card.
   */
  readonly selectInspectionType = output<InspectionType>();
}
