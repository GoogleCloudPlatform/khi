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

import { Component, computed, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { LogType, Severity } from 'src/app/store/domain/style';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';

/**
 * `TypeSeverityComponent` displays a visual badge representing the severity and type of a log entry.
 * It applies dynamic styles using background and foreground colors loaded from the session.
 */
@Component({
  selector: 'khi-type-severity',
  standalone: true,
  templateUrl: './type-severity.component.html',
  styleUrls: ['./type-severity.component.scss'],
  imports: [CommonModule, MatIconModule, MatTooltipModule],
})
export class TypeSeverityComponent {
  /**
   * The log type configuration containing visual styles and label.
   */
  logType = input<LogType | null>(null);

  /**
   * The severity configuration containing visual styles and label.
   */
  severity = input<Severity | null>(null);

  /**
   * Dynamically computed background and color style mapping for the log type badge.
   */
  protected readonly typeStyle = computed(() => {
    const t = this.logType();
    if (!t) return {};
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
   * Dynamically computed background and color style mapping for the severity badge.
   */
  protected readonly severityStyle = computed(() => {
    const s = this.severity();
    if (!s) return {};
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
}
