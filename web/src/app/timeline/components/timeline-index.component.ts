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

import { Component, computed, input, output } from '@angular/core';
import { Timeline } from 'src/app/store/domain/timeline';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatDividerModule } from '@angular/material/divider';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimelineHighlight, TimelineHighlightType } from './interaction-model';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';
import { StyleStoreLike } from 'src/app/store/domain/style-store';

export interface TreeGuideViewModel {
  readonly level: number;
  readonly isParent: boolean;
  readonly isLastChild: boolean;
}

interface TimelineIndexViewModel {
  /** The resource timeline data associated with this row. */
  timeline: Timeline;
  /** The display name of the resource. */
  name: string;
  /** Optional icon name to display in the legend for specific layers (e.g., 'workspaces', 'folder'). */
  legendIcon?: string;
  /** Whether the legend should be visible. */
  legendVisible: boolean;
  /** The full resource path. It's unique for each timeline and used as the identifier. */
  resourcePath: string;
  /** Secondary label text, currently only used for showing the resource group/version for Kind layer rows. */
  subLabel: string;
  /** The name of the layer this row belongs to (e.g., 'cluster', 'node', 'pod'). */
  layerName: string;
  /** The depth level of the layer (0-indexed or 1-indexed, e.g., Kind=1, Namespace=2). */
  layer: number;
  /** Sibling level guide configurations. */
  levels: TreeGuideViewModel[];
  /** Whether this row is the last child of its parent (nesting level decreases next). */
  isLastChild: boolean;
  /**
   * isNextChildren is true when the next element is a child of this resource.
   * This is used to render a drop shadow or visual grouping indicator.
   */
  isNextChildren: boolean;
  /** Whether this timeline has child timelines. */
  hasChildren: boolean;
  /** Whether this timeline is currently collapsed. */
  isCollapsed: boolean;
  /** CSS classes to apply to the row container. */
  containerClasses: string[];
  /** Custom style values mapped to CSS custom properties (variables) for inline styling. */
  style: Record<string, string>;
}

/**
 * Component that renders the index (left sidebar) of the timeline view.
 * Displays resource names, hierarchy indicators, and handles selection/hover interactions for given timelines.
 */
@Component({
  selector: 'khi-timeline-index',
  templateUrl: './timeline-index.component.html',
  styleUrl: './timeline-index.component.scss',
  imports: [
    CommonModule,
    MatTooltipModule,
    MatMenuModule,
    MatDividerModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
})
export class TimelineIndexComponent {
  /** The list of resource timelines to display in the index. */
  timelines = input<ReadonlyDomainElement<Timeline[]>>([]);

  /** Current set of collapsed timeline IDs. */
  collapsedTimelineIds = input<ReadonlySet<number>>(new Set());

  /** Current highlight state for timelines, keyed by timeline ID. */
  highlights = input<TimelineHighlight>({});

  /** The StyleStore containing all color and layout styling definitions. */
  styleStore = input<StyleStoreLike>();

  /** Computed view models for rendering the timeline index rows. */
  timelineVMs = computed<TimelineIndexViewModel[]>(() => {
    this.styleStore()?.stylesUpdated?.();
    return this.toViewModelType(this.timelines());
  });

  /** Emits the timeline when the user hovers over a row. */
  hoverOnTimeline = output<Timeline>();

  /** Emits the timeline when the user clicks on a row. */
  clickOnTimeline = output<Timeline>();

  /** Emits the timeline when requesting to toggle its collapse state. */
  toggleCollapseTimeline = output<Timeline>();

  /** Emits the timeline when requesting to expand its direct children timelines. */
  expandChildren = output<Timeline>();

  /** Emits the timeline when requesting to collapse its direct children timelines. */
  collapseChildren = output<Timeline>();

  /** Emits the timeline when requesting to exclude it. */
  excludeTimeline = output<Timeline>();

  /** Emits the timeline type label when requesting to exclude all timelines of that type. */
  excludeTimelineType = output<string>();

  /**
   * Handles mouse over events on a timeline row.
   * @param timeline - The timeline that is being hovered.
   */
  mouseOverTimeline(timeline: Timeline) {
    this.hoverOnTimeline.emit(timeline);
  }

  /**
   * Handles click events on a timeline row.
   * @param timeline - The timeline that was clicked.
   */
  clickTimeline(timeline: Timeline) {
    this.clickOnTimeline.emit(timeline);
  }

  /**
   * Converts raw Timeline objects into ViewModel objects for rendering.
   * Calculates styles, classes, and display properties based on the current state.
   *
   * @param timelines - The list of Timeline objects to convert.
   * @returns An array of TimelineIndexViewModel objects.
   */
  toViewModelType(
    timelines: ReadonlyDomainElement<Timeline[]>,
  ): TimelineIndexViewModel[] {
    const highlights = this.highlights();
    const styleStore = this.styleStore();
    const collapsedSet = this.collapsedTimelineIds();
    return timelines.map((timeline, i, arr) => {
      const timelineType =
        styleStore?.getTimelineType(timeline.type.id) ?? timeline.type;
      const nextTimeline = arr[i + 1];
      const isNextChildren =
        nextTimeline && nextTimeline.layer > timeline.layer;
      const containerClasses = [timelineType.label];
      if (isNextChildren) {
        containerClasses.push('is-next-children');
      }
      const highlight = highlights[timeline.id];
      switch (highlight) {
        case TimelineHighlightType.Selected:
          containerClasses.push('selected');
          break;
        case TimelineHighlightType.Hovered:
          containerClasses.push('hovered');
          break;
        case TimelineHighlightType.ChildrenOfSelected:
          containerClasses.push('children-of-selected');
          break;
      }
      const bg = timelineType.backgroundColor;
      const fg = timelineType.foregroundColor;
      const chipBg = timelineType.typeChipBackgroundColor;
      const height = timelineType.height * BASE_ROW_HEIGHT;
      const style: Record<string, string> = {
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
          timelineType.typeChipForegroundColor.r,
          timelineType.typeChipForegroundColor.g,
          timelineType.typeChipForegroundColor.b,
          timelineType.typeChipForegroundColor.a,
        ]),
        '--timeline-height': `${height}px`,
        '--timeline-layer': `${timeline.layer}`,
      };
      const isLastChild = !nextTimeline || nextTimeline.layer < timeline.layer;

      const levels: TreeGuideViewModel[] = [];
      for (let d = 0; d < timeline.layer; d++) {
        const isParent = d === timeline.layer - 1;
        let hasFutureSibling = false;
        for (let j = i + 1; j < timelines.length; j++) {
          const next = timelines[j];
          if (next.layer === d + 1) {
            hasFutureSibling = true;
            break;
          }
          if (next.layer < d + 1) {
            break;
          }
        }
        if (isParent || hasFutureSibling) {
          levels.push({
            level: d,
            isParent: isParent,
            isLastChild: isParent ? !hasFutureSibling : false,
          });
        }
      }
      return {
        timeline: timeline,
        resourcePath: timeline.id.toString(),
        name: timeline.name,
        legendIcon: timeline.type.icon || undefined,
        legendVisible: !!timeline.type.icon,
        subLabel: '',
        layerName: timeline.type.label,
        layer: timeline.layer,
        levels: levels,
        isLastChild: isLastChild,
        isNextChildren: isNextChildren,
        hasChildren: timeline.childrenCount > 0,
        isCollapsed: collapsedSet.has(timeline.id),
        containerClasses: containerClasses,
        style: style,
      } as TimelineIndexViewModel;
    });
  }
}
