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

import {
  afterRenderEffect,
  Component,
  computed,
  input,
  output,
  viewChild,
} from '@angular/core';
import { MatDialogModule } from '@angular/material/dialog';
import { MatStepper, MatStepperModule } from '@angular/material/stepper';
import {
  InspectionFeature,
  InspectionType,
} from 'src/app/common/schema/api-types';
import { FeatureSelectorComponent } from 'src/app/dialogs/new-inspection/components/feature-selector.component';
import { InspectionTypeSelectorComponent } from 'src/app/dialogs/new-inspection/components/inspection-type-selector.component';
import { ParameterInputStepComponent } from 'src/app/dialogs/new-inspection/components/parameter-input-step.component';
import { ParameterStore } from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  NewInspectionStepIndex,
  ParameterStepViewModel,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { buildTaskGraphDebugUrl } from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

/**
 * Layout dumb component that displays the complete new inspection wizard dialog layout.
 */
@Component({
  selector: 'khi-new-inspection-layout',
  templateUrl: './new-inspection-layout.component.html',
  styleUrl: './new-inspection-layout.component.scss',
  imports: [
    MatDialogModule,
    MatStepperModule,
    InspectionTypeSelectorComponent,
    FeatureSelectorComponent,
    ParameterInputStepComponent,
  ],
})
export class NewInspectionLayoutComponent {
  protected readonly NewInspectionStepIndex = NewInspectionStepIndex;

  private readonly stepper = viewChild<MatStepper>('stepper');

  constructor() {
    // MatStepper in linear mode blocks [selectedIndex] changes during the same change
    // detection pass where preceding steps become completed. Synchronizing after render
    // ensures programmatic multi-step transitions apply once step completion states settle.
    afterRenderEffect(() => {
      const index = this.selectedStepIndex();
      const stepper = this.stepper();
      if (stepper && stepper.selectedIndex !== index) {
        stepper.selectedIndex = index;
      }
    });
  }

  /**
   * The currently active step index in the stepper.
   */
  readonly selectedStepIndex = input.required<NewInspectionStepIndex>();

  /**
   * The available inspection types, or null while loading.
   */
  readonly inspectionTypes = input.required<readonly InspectionType[] | null>();

  /**
   * The currently selected inspection type, or null if none is chosen.
   */
  readonly currentInspectionType = input.required<InspectionType | null>();

  /**
   * The list of features available for the selected inspection type.
   */
  readonly features = input.required<readonly InspectionFeature[]>();

  /**
   * Store holding current, default, and validated parameter values.
   */
  readonly parameterStore = input.required<ParameterStore>();

  /**
   * Indicates whether at least one feature is currently selected/enabled.
   */
  readonly hasEnabledFeatures = computed(() =>
    this.features().some((f) => f.enabled),
  );

  /**
   * The view model for the parameter input step, or null while resolving.
   */
  readonly parameterViewModel = input.required<ParameterStepViewModel | null>();

  /**
   * URL to open the Task Graph Diagnostics page for the current inspection type and enabled features.
   */
  readonly taskGraphDebugUrl = computed(() =>
    buildTaskGraphDebugUrl(this.currentInspectionType(), this.features()),
  );

  /**
   * Emitted when the user changes or navigates to a new step.
   */
  readonly stepChange = output<NewInspectionStepIndex>();

  /**
   * Emitted when an inspection type is clicked/selected.
   */
  readonly selectInspectionType = output<InspectionType>();

  /**
   * Emitted when a feature is toggled by its ID.
   */
  readonly toggleFeature = output<string>();

  /**
   * Emitted when the run inspection button is clicked.
   */
  readonly runInspection = output<void>();
}
