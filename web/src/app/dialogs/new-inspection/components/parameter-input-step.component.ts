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
import { DecimalPipe } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { GroupParameterComponent } from 'src/app/dialogs/new-inspection/components/group-parameter.component';
import { JobCommandComponent } from 'src/app/dialogs/new-inspection/components/job-command.component';
import { EstimatedCountPreset } from 'src/app/common/schema/metadata-types';
import {
  ParameterStepViewModel,
  TotalEstimatedLogsSeverity,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';

/**
 * Dumb component that renders the parameter input step of the new inspection wizard.
 */
@Component({
  selector: 'khi-new-inspection-parameter-input-step',
  templateUrl: './parameter-input-step.component.html',
  styleUrl: './parameter-input-step.component.scss',
  imports: [
    DecimalPipe,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    KHIIconRegistrationModule,
    GroupParameterComponent,
    JobCommandComponent,
  ],
})
export class ParameterInputStepComponent {
  protected readonly EstimatedCountPreset = EstimatedCountPreset;
  protected readonly TotalEstimatedLogsSeverity = TotalEstimatedLogsSeverity;

  /**
   * Holds the view model containing forms, queries, execution plan, and metadata.
   */
  readonly parameterViewModel = input.required<ParameterStepViewModel | null>();

  /**
   * The count of input parameter fields currently containing validation errors.
   */
  readonly errorFieldCount = input.required<number>();

  /**
   * The count of input parameter fields currently resolving asynchronously.
   */
  readonly pendingFieldCount = input.required<number>();

  /**
   * Indicates whether the run inspection button should be disabled.
   */
  readonly isRunButtonDisabled = input.required<boolean>();

  /**
   * Emitted when the user clicks the run inspection button.
   */
  readonly runInspection = output<void>();
}
