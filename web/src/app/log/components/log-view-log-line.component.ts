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

import { Component, input, output } from '@angular/core';
import { LogEntry } from 'src/app/store/log';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { TimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';

/**
 * `LogViewLogLineComponent` renders a single log entry row within the virtualized log list.
 * It visualizes the log's type, severity (with appropriate color-coding), timestamp, and summary.
 * Uses Angular signals for reactive inputs and outputs.
 */
@Component({
  selector: 'khi-log-view-log-line',
  templateUrl: './log-view-log-line.component.html',
  styleUrls: ['./log-view-log-line.component.scss'],
  imports: [CommonModule, MatTooltipModule, TimestampFormatPipe],
})
export class LogViewLogLineComponent {
  /**
   * The LogEntry to show in this line.
   */
  readonly log = input.required<LogEntry>();

  /**
   * Whether this log line is currently selected.
   */
  readonly selected = input<boolean>(false);

  /**
   * Whether this log line is currently highlighted.
   */
  readonly highlighted = input<boolean>(false);

  /**
   * An event triggered when user's mouse cursor hover on this line.
   */
  readonly lineHover = output<LogEntry>();

  /**
   * Emits the clicked `LogEntry` when the user selects this log line.
   * This is typically used by the parent component to update the detailed view state.
   */
  readonly lineClick = output<LogEntry>();

  /**
   * Internal click handler that triggers the `lineClick` output signal.
   */
  protected onClick() {
    this.lineClick.emit(this.log());
  }

  /**
   * Internal hover handler that triggers the `lineHover` output signal.
   */
  protected onHover() {
    this.lineHover.emit(this.log());
  }
}
