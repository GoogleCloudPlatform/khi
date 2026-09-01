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
import { LogStore } from 'src/app/store/domain/log-store';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { TimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';
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
   * The ID of the log entry to show in this line.
   */
  public readonly logId = input.required<number>();

  /**
   * The store containing log data.
   */
  public readonly logStore = input.required<LogStore>();

  /**
   * Whether this log line is currently selected.
   */
  public readonly selected = input<boolean>(false);

  /**
   * Whether this log line is currently highlighted.
   */
  public readonly highlighted = input<boolean>(false);

  /**
   * The human-readable summary of the log.
   */
  protected readonly summary = computed(() =>
    this.logStore().getSummary(this.logId()),
  );

  /**
   * The timestamp of the log in milliseconds for display.
   */
  protected readonly timestampMs = computed(() =>
    BigIntTimeUtil.NsToNumberMs(this.logStore().getTimestamp(this.logId())),
  );

  /**
   * The log type metadata.
   */
  protected readonly logType = computed(() =>
    this.logStore().getLogType(this.logId()),
  );

  /**
   * The severity metadata.
   */
  protected readonly severity = computed(() =>
    this.logStore().getSeverity(this.logId()),
  );

  /**
   * Dynamic background and text styling for the log type badge.
   */
  protected readonly typeStyle = computed(() => {
    const t = this.logType();
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
    const s = this.severity();
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
   * Emits the log ID when the user hovers over this line.
   */
  public readonly lineHover = output<number>();

  /**
   * Emits the log ID when the user selects this log line.
   */
  public readonly lineClick = output<number>();

  /**
   * Internal click handler that triggers the `lineClick` output signal.
   */
  protected onClick() {
    this.lineClick.emit(this.logId());
  }

  /**
   * Internal hover handler that triggers the `lineHover` output signal.
   */
  protected onHover() {
    this.lineHover.emit(this.logId());
  }
}
