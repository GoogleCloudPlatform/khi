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
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDialogModule } from '@angular/material/dialog';
import { MatTabsModule } from '@angular/material/tabs';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  RevisionStateStyleOverrideViewModel,
  TimelineTypeStyleOverrideViewModel,
  TimelineTypeOverrideEvent,
  LogTypeStyleOverrideViewModel,
  LogTypeOverrideEvent,
} from 'src/app/dialogs/style-override/types/style-override-viewmodel';
import { RevisionStateOverrideListComponent } from 'src/app/dialogs/style-override/components/revision-state-override-list.component';
import { TimelineTypeOverrideListComponent } from 'src/app/dialogs/style-override/components/timeline-type-override-list.component';
import { LogTypeOverrideListComponent } from 'src/app/dialogs/style-override/components/log-type-override-list.component';

/**
 * Dumb component that presents a tabbed panel for overriding styles (Revision States, Timeline Types).
 */
@Component({
  selector: 'khi-style-override-layout',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatDialogModule,
    MatTabsModule,
    KHIIconRegistrationModule,
    RevisionStateOverrideListComponent,
    TimelineTypeOverrideListComponent,
    LogTypeOverrideListComponent,
  ],
  templateUrl: './style-override-layout.component.html',
  styleUrls: ['./style-override-layout.component.scss'],
})
export class StyleOverrideLayoutComponent {
  /** List of revision state view models to display. */
  readonly revisionStates =
    input.required<RevisionStateStyleOverrideViewModel[]>();

  /** List of timeline type view models to display. */
  readonly timelineTypes =
    input.required<TimelineTypeStyleOverrideViewModel[]>();

  /** List of log type view models to display. */
  readonly logTypes = input.required<LogTypeStyleOverrideViewModel[]>();

  /** Emitted when a revision state's color is overridden. */
  readonly revisionStateColorChange = output<{
    readonly id: number;
    readonly hexColor: string;
  }>();

  /** Emitted when a revision state's color override is reset. */
  readonly revisionStateResetColor = output<number>();

  /** Emitted when a timeline type style property is overridden. */
  readonly timelineTypePropertyChange = output<TimelineTypeOverrideEvent>();

  /** Emitted when a timeline type's color override is reset. */
  readonly timelineTypeResetColor = output<number>();

  /** Emitted when a log type style property is overridden. */
  readonly logTypePropertyChange = output<LogTypeOverrideEvent>();

  /** Emitted when a log type's color override is reset. */
  readonly logTypeResetColor = output<number>();
}
