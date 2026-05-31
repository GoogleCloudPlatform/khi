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
  Component,
  HostListener,
  input,
  model,
  output,
  inject,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBar } from '@angular/material/snack-bar';
import { OverlayModule } from '@angular/cdk/overlay';
import { CelInputComponent } from 'src/app/timeline-toolbar/components/cel-input.component';
import { ToolbarSettingsComponent } from 'src/app/timeline-toolbar/components/toolbar-settings.component';

/**
 * Provides an advanced toolbar for the timeline view featuring two-row CEL expression text fields.
 */
@Component({
  selector: 'khi-timeline-toolbar-advanced',
  templateUrl: './toolbar-advanced.component.html',
  styleUrls: ['./toolbar-advanced.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    CelInputComponent,
    ToolbarSettingsComponent,
    MatButtonModule,
    OverlayModule,
    MatTooltipModule,
  ],
})
export class ToolbarAdvancedComponent {
  private readonly snackbar = inject(MatSnackBar);

  /**
   * Signal managing the open/closed state of the settings popover.
   */
  protected readonly settingsPopupOpen = signal(false);

  /**
   * Holds the two-way bound timezone shift in hours.
   */
  readonly timezoneShift = model(0);

  /**
   * Specifies whether a log or timeline is currently not selected.
   */
  readonly logOrTimelineNotSelected = input(true);

  /**
   * Validation error message for the timeline CEL filter.
   */
  readonly timelineCelError = input('');

  /**
   * Validation error message for the log CEL filter.
   */
  readonly logCelError = input('');

  /**
   * Holds the two-way bound option to hide timelines without matching logs.
   */
  readonly hideTimelinesWithoutMatchingLogs = model(false);

  /**
   * Holds the two-way bound timeline CEL filter string.
   */
  readonly timelineCelFilter = model('');

  /**
   * Holds the two-way bound log CEL filter string.
   */
  readonly logCelFilter = model('');

  /**
   * Emits an event to request drawing the architecture diagram.
   */
  readonly drawDiagram = output<void>();

  /**
   * Emits an event to switch to standard mode.
   */
  readonly switchToStandard = output<void>();

  /**
   * Handles the change event for the timezone shift input.
   */
  public onTimezoneshiftCommit(event: Event): void {
    const value = +(event.target as HTMLInputElement).value;
    this.timezoneShift.set(isNaN(value) ? 0 : value);
  }

  /**
   * Intercepts standard browser search shortcut to notify users about virtual rendering limitations.
   */
  @HostListener('window:keydown', ['$event'])
  protected interceptBrowserSearch(event: KeyboardEvent): void {
    if (event.key === 'f' && (event.ctrlKey || event.metaKey)) {
      this.snackbar.open(
        'In-browser search may not work on KHI because elements outside the visible area are not rendered. Please use the search text field on the toolbar instead.',
        'OK',
      );
    }
  }
}
