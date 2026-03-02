/**
 * Copyright 2024 Google LLC
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

import { Component, computed, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LogContentHeaderComponent } from './log-content-header.component';
import { HighlightModule } from 'ngx-highlightjs';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltip } from '@angular/material/tooltip';
import { LogEntry } from 'src/app/store/log';
import { ResourceTimeline } from 'src/app/store/timeline';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatButtonModule } from '@angular/material/button';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { Clipboard, ClipboardModule } from '@angular/cdk/clipboard';

/**
 * View model aggregating the full detailed data required to render the log content and header.
 */
export interface LogContentViewModel {
  logEntry: LogEntry | null;
  logBody: string;
  parsedLogBody: unknown;
  referencedResourcePaths: string[];
}

/**
 * Component responsible for displaying the detailed body of a log entry.
 * Provides actions such as copying the raw log content and copying a Cloud Logging query
 * for the specific log entry.
 */
@Component({
  selector: 'khi-log-content',
  templateUrl: './log-content.component.html',
  styleUrls: ['./log-content.component.scss'],
  imports: [
    CommonModule,
    LogContentHeaderComponent,
    HighlightModule,
    MatIconModule,
    MatTooltip,
    KHIIconRegistrationModule,
    MatButtonModule,
    MatSnackBarModule,
    ClipboardModule,
  ],
})
export class LogContentComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  /**
   * The aggregated view model containing the log entry, body, and resolved references.
   */
  public readonly vm = input<LogContentViewModel | null>(null);

  /**
   * The timezone shift to apply to the timestamp.
   */
  public timezoneShift = input<number>(0);

  /**
   * Output emitted when a resource timeline is clicked from the reference list.
   */
  public resourceSelected = output<string>();

  /**
   * Output emitted when a resource timeline is hovered from the reference list.
   */
  public resourceHighlighted = output<string>();

  /**
   * Input tracking the currently selected timeline to visually indicate selection state
   * in the resource reference list.
   */
  public selectedTimeline = input<ResourceTimeline | null>(null);

  private readonly timestampString = computed(() => {
    const parsed = this.vm()?.parsedLogBody as
      | { [key: string]: string }
      | undefined;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed['timestamp'] ?? null;
    }
    return null;
  });

  /**
   * Determines if the "Copy Query" button should be visible.
   * True only if a valid timestamp can be extracted from the loaded log body.
   */
  protected readonly showCopyQueryButton = computed(() => {
    return this.timestampString() !== null;
  });

  /**
   * Copies the loaded log body text to the clipboard and displays a notification.
   */
  copyLog() {
    const logBody = this.vm()?.logBody;
    if (!logBody) {
      return;
    }
    this.showCopySnackbarMessage(this.clipboard.copy(logBody));
  }

  /**
   * Copies a Cloud Logging query string uniquely identifying this log to the clipboard.
   * Extracts the insertId and timestamp from the log body to build the query.
   */
  copyLogQuery() {
    const log = this.vm()?.logEntry;
    const timestampString = this.timestampString();
    if (!log || !timestampString) {
      return;
    }
    this.showCopySnackbarMessage(
      this.clipboard.copy(`(
-- Log query for "${log.summary}"
insertId="${log.insertId}"
timestamp="${timestampString}"
)`),
    );
  }

  /**
   * Displays a snackbar notification indicating the result of a copy action.
   * @param success Whether the copy to clipboard operation was successful.
   */
  private showCopySnackbarMessage(success: boolean) {
    this.snackBar.open(success ? 'Copied!' : 'Copy failed', undefined, {
      duration: 1000,
    });
  }
}
