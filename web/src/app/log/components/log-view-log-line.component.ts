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

import { Component, computed, input, output } from '@angular/core';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { TimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';

import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';

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
  readonly log = input.required<ReadonlyDomainElement<Log>>();

  /**
   * Whether this log line is currently selected.
   */
  readonly selected = input<boolean>(false);

  /**
   * Whether this log line is currently highlighted.
   */
  readonly highlighted = input<boolean>(false);

  /**
   * Dynamic background and text styling for the log type badge.
   */
  protected readonly typeStyle = computed(() => {
    const t = this.log().logType;
    const bg = RendererConvertUtil.hdrColorToCSSColor([
      t.backgroundColor.r,
      t.backgroundColor.g,
      t.backgroundColor.b,
      t.backgroundColor.a,
    ]);
    const fg = RendererConvertUtil.hdrColorToCSSColor([
      t.foregroundColor.r,
      t.foregroundColor.g,
      t.foregroundColor.b,
      t.foregroundColor.a,
    ]);
    return { 'background-color': bg, color: fg };
  });

  /**
   * Dynamic background and text styling for the severity indicator badge.
   */
  protected readonly severityStyle = computed(() => {
    const s = this.log().severity;
    const bg = RendererConvertUtil.hdrColorToCSSColor([
      s.backgroundColor.r,
      s.backgroundColor.g,
      s.backgroundColor.b,
      s.backgroundColor.a,
    ]);
    const fg = RendererConvertUtil.hdrColorToCSSColor([
      s.foregroundColor.r,
      s.foregroundColor.g,
      s.foregroundColor.b,
      s.foregroundColor.a,
    ]);
    return { 'background-color': bg, color: fg };
  });

  /**
   * An event triggered when user's mouse cursor hover on this line.
   */
  readonly lineHover = output<ReadonlyDomainElement<Log>>();

  /**
   * Emits the clicked `LogEntry` when the user selects this log line.
   * This is typically used by the parent component to update the detailed view state.
   */
  readonly lineClick = output<ReadonlyDomainElement<Log>>();

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
