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

import { CommonModule } from '@angular/common';
import { Component, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimelineType } from 'src/app/store/domain/style';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';

/**
 * Represents a view model for a single node in the parent resource hierarchy path.
 */
export interface ResourcePathNodeViewModel {
  readonly id: number;
  readonly label: string;
  readonly type: TimelineType;
}

/**
 * Converts a TimelineType into CSS custom property styles for background and chip colors.
 */
export function getTimelineStyle(type: TimelineType): Record<string, string> {
  const bg = type.backgroundColor;
  const fg = type.foregroundColor;
  const chipBg = type.typeChipBackgroundColor;
  const chipFg = type.typeChipForegroundColor;
  return {
    '--timeline-bg-color': RendererConvertUtil.hdrColorToCSSColor([
      bg.r,
      bg.g,
      bg.b,
      bg.a,
    ]),
    '--timeline-fg-color': RendererConvertUtil.hdrColorToCSSColor([
      fg.r,
      fg.g,
      fg.b,
      fg.a,
    ]),
    '--timeline-chip-bg-color': RendererConvertUtil.hdrColorToCSSColor([
      chipBg.r,
      chipBg.g,
      chipBg.b,
      chipBg.a,
    ]),
    '--timeline-chip-fg-color': RendererConvertUtil.hdrColorToCSSColor([
      chipFg.r,
      chipFg.g,
      chipFg.b,
      chipFg.a,
    ]),
  };
}

/**
 * `ResourceHierarchyOverlayComponent` renders the parent hierarchy structure of a resource.
 * It displays a vertical tree list of ancestor nodes (including type chips and icons).
 */
@Component({
  selector: 'khi-resource-hierarchy-overlay',
  standalone: true,
  templateUrl: './resource-hierarchy-overlay.component.html',
  styleUrl: './resource-hierarchy-overlay.component.scss',
  imports: [CommonModule, MatIconModule, KHIIconRegistrationModule],
})
export class ResourceHierarchyOverlayComponent {
  /**
   * The list of ancestor path nodes representing the resource hierarchy.
   */
  readonly pathNodes = input<ResourcePathNodeViewModel[]>([]);

  /**
   * Converts a TimelineType into CSS custom property styles for background and chip colors.
   */
  public getTimelineStyle(type: TimelineType): Record<string, string> {
    return getTimelineStyle(type);
  }
}
