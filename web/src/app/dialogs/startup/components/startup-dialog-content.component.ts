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
import { InspectionListComponent } from './inspection-list.component';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';

/**
 * Content component for the Startup Dialog V2.
 * Displays the inspection list.
 */
@Component({
  selector: 'khi-startup-dialog-content',
  imports: [InspectionListComponent],
  templateUrl: './startup-dialog-content.component.html',
  styleUrls: ['./startup-dialog-content.component.scss'],
})
export class StartupDialogContentComponent {
  /** List of inspection items to display. */
  public readonly items = input.required<InspectionListItemViewModel[]>();

  /** Whether the list is loading. */
  public readonly isLoading = input<boolean>(false);

  /** Whether the application is in viewer mode (read-only). */
  public readonly isViewerMode = input<boolean>(false);

  /** Emitted when the user clicks to create a new inspection from the empty state. */
  public readonly createNewInspection = output<void>();

  /** Emitted when an inspection result is opened. */
  public readonly openInspectionResult = output<string>();

  /** Emitted when inspection metadata is opened. */
  public readonly openInspectionMetadata = output<string>();

  /** Emitted when an inspection is cancelled. */
  public readonly cancelInspection = output<string>();

  /** Emitted when an inspection result is downloaded. */
  public readonly downloadInspectionResult = output<string>();

  /** Emitted when an inspection title is changed. */
  public readonly changeInspectionTitle = output<{
    id: string;
    changeTo: string;
  }>();
}
