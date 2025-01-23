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

import { Component, EventEmitter, Input, Output } from '@angular/core';
import { LogEntry } from '../store/log';

/**
 * A line of log in log view list.
 */
@Component({
  selector: 'khi-log-view-log-line',
  templateUrl: './log-view-log-line.component.html',
  styleUrls: ['./log-view-log-line.component.sass'],
})
export class LogViewLogLineComponent {
  /**
   * The LogEntry to show in this line.
   */
  @Input()
  log!: LogEntry;

  /**
   * An event triggered when user's mouse curosr hover on this line.
   */
  @Output()
  lineHover: EventEmitter<LogEntry> = new EventEmitter();

  /**
   * An event triggered when user clicked this log line.
   */
  @Output()
  lineClick: EventEmitter<LogEntry> = new EventEmitter();

  onClick() {
    this.lineClick.emit(this.log);
  }

  onHover() {
    this.lineHover.emit(this.log);
  }
}
