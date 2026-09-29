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

import { Component, computed, input, output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { BreaklinePipe } from 'src/app/common/breakline.pipe';
import { InspectionFeature } from 'src/app/common/schema/api-types';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Dumb component for selecting features (log types / data sources) in Step 2 of the new inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-feature-selector',
  templateUrl: './feature-selector.component.html',
  styleUrls: ['./feature-selector.component.scss'],
  imports: [
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    BreaklinePipe,
  ],
})
export class FeatureSelectorComponent {
  /**
   * The list of inspection features to display.
   */
  readonly features = input.required<readonly InspectionFeature[]>();

  /**
   * Whether at least one feature is enabled, allowing progression to the next step.
   */
  readonly hasEnabledFeatures = computed(() =>
    this.features().some((f) => f.enabled),
  );

  /**
   * Emits the feature ID when the user clicks a feature card to toggle its enabled status.
   */
  readonly toggleFeature = output<string>();

  /**
   * Emits when the user clicks the Next button to proceed to the next step.
   */
  readonly nextStep = output<void>();
}
