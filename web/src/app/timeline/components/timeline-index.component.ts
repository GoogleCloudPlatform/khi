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
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { CommonModule } from '@angular/common';
import {
  ParentRelationshipMetadata,
  ParentRelationshipMetadataType,
} from 'src/app/generated';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimelineHighlight, TimelineHighlightType } from './interaction-model';

interface TimelineIndexViewModel {
  /** The resource timeline data associated with this row. */
  timeline: ResourceTimeline;
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
  /** Metadata describing the parent relationship (e.g., OwnerReference, Label). */
  relationshipMetadata: ParentRelationshipMetadataType;
  /**
   * isNextChildren is true when the next element is a child of this resource.
   * This is used to render a drop shadow or visual grouping indicator.
   */
  isNextChildren: boolean;
  /** CSS classes to apply to the row container. */
  containerClasses: string[];
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
    MatIconModule,
    KHIIconRegistrationModule,
  ],
})
export class TimelineIndexComponent {
  /** Map of TimelineLayer values to their corresponding Material Symbol icon names. */
  readonly LayerIcons: { [key in TimelineLayer]?: string } = {
    [TimelineLayer.Kind]: 'workspaces',
    [TimelineLayer.Namespace]: 'folder',
  };

  /** The list of resource timelines to display in the index. */
  timelines = input<ResourceTimeline[]>([]);

  /** Current highlight state for timelines, keyed by timeline ID. */
  highlights = input<TimelineHighlight>({});

  /** Computed view models for rendering the timeline index rows. */
  timelineVMs = computed<TimelineIndexViewModel[]>(() => {
    return this.toViewModelType(this.timelines());
  });

  /** Emits the timeline when the user hovers over a row. */
  hoverOnTimeline = output<ResourceTimeline>();

  /** Emits the timeline when the user clicks on a row. */
  clickOnTimeline = output<ResourceTimeline>();

  /**
   * Handles mouse over events on a timeline row.
   * @param timeline - The timeline that is being hovered.
   */
  mouseOverTimeline(timeline: ResourceTimeline) {
    this.hoverOnTimeline.emit(timeline);
  }

  /**
   * Handles click events on a timeline row.
   * @param timeline - The timeline that was clicked.
   */
  clickTimeline(timeline: ResourceTimeline) {
    this.clickOnTimeline.emit(timeline);
  }

  /**
   * Converts raw ResourceTimeline objects into ViewModel objects for rendering.
   * Calculates styles, classes, and display properties based on the current state.
   *
   * @param timelines - The list of ResourceTimeline objects to convert.
   * @returns An array of TimelineIndexViewModel objects.
   */
  toViewModelType(timelines: ResourceTimeline[]): TimelineIndexViewModel[] {
    const highlights = this.highlights();
    return timelines.map((timeline, i, arr) => {
      const nextTimeline = arr[i + 1];
      const isNextChildren =
        nextTimeline && nextTimeline.layer > timeline.layer;
      const containerClasses = [timeline.layerName];
      if (isNextChildren) {
        containerClasses.push('is-next-children');
      }
      const highlight = highlights[timeline.timelineId];
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
      return {
        timeline: timeline,
        resourcePath: timeline.resourcePath,
        name: timeline.name,
        legendIcon: this.LayerIcons[timeline.layer],
        legendVisible:
          timeline.layer === TimelineLayer.Kind ||
          timeline.layer === TimelineLayer.Namespace,
        subLabel:
          timeline.layer === TimelineLayer.Kind
            ? timeline.resourcePath.split('#')[0]
            : '',
        layerName: timeline.layerName,
        relationshipMetadata:
          ParentRelationshipMetadata[timeline.parentRelationship],
        isNextChildren: isNextChildren,
        containerClasses: containerClasses,
      };
    });
  }
}
