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
import { InspectionListItemComponent } from './inspection-list-item.component';
import {
  InspectionListItemViewModel,
  InspectionTitleChangeRequest,
} from '../types/inspection-activity.model';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Component for displaying a list of inspection activities.
 * This is a dumb component that delegates rendering to InspectionListItemComponent.
 */
@Component({
  selector: 'khi-inspection-list',
  imports: [
    InspectionListItemComponent,
    MatIconModule,
    MatProgressSpinnerModule,
    MatButtonModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './inspection-list.component.html',
  styleUrls: ['./inspection-list.component.scss'],
})
export class InspectionListComponent {
  /** The list of inspection activities to display. */
  public readonly items = input.required<InspectionListItemViewModel[]>();

  /** Whether the list is loading. */
  public readonly isLoading = input<boolean>(false);

  /** Whether the application is in viewer mode (read-only). */
  public readonly isViewerMode = input<boolean>(false);

  /** Emits when the user clicks to create a new inspection from the empty state. */
  public readonly createNewInspection = output<void>();

  /** Emits the ID of the inspection to open the result for. */
  public readonly openInspectionResult = output<string>();

  /** Emits the ID of the inspection to open the metadata for. */
  public readonly openInspectionMetadata = output<string>();

  /** Emits the ID of the inspection to cancel. */
  public readonly cancelInspection = output<string>();

  /** Emits the ID of the inspection to download the result for. */
  public readonly downloadInspectionResult = output<string>();

  /** Emits the ID and the new title when an inspection title is changed. */
  public readonly changeInspectionTitle =
    output<InspectionTitleChangeRequest>();
}
