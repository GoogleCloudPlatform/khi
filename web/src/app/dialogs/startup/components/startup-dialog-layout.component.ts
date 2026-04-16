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
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { StartupSideMenuComponent } from './startup-side-menu.component';
import { StartupDialogContentComponent } from './startup-dialog-content.component';
import { SidebarLink } from '../types/startup-side-menu.types';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';

/**
 * Layout component for the Startup Dialog V2.
 * Directly arranges the side menu and the content area.
 */
@Component({
  selector: 'khi-startup-dialog-layout',
  imports: [
    StartupSideMenuComponent,
    StartupDialogContentComponent,
    MatProgressSpinnerModule,
  ],
  templateUrl: './startup-dialog-layout.component.html',
  styleUrls: ['./startup-dialog-layout.component.scss'],
})
export class StartupDialogLayoutComponent {
  /** The current version of the application to be displayed in the sidebar. */
  public readonly version = input.required<string>();

  /** The list of links to be displayed in the sidebar footer. */
  public readonly links = input.required<SidebarLink[]>();

  /** List of inspection items to display. */
  public readonly items = input.required<InspectionListItemViewModel[]>();

  /** Whether the tasks are loading. */
  public readonly isLoading = input<boolean>(false);

  /** Whether the application is in viewer mode (read-only). */
  public readonly isViewerMode = input<boolean>(false);

  /** Emitted when the user clicks the 'New Investigation' button. */
  public readonly newInvestigation = output<void>();

  /** Emitted when the user clicks the 'Open .khi file' button. */
  public readonly openKhiFile = output<void>();

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
