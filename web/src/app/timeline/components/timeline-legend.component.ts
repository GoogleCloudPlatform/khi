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
import { Component, computed, input, model } from '@angular/core';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import {
  LogType,
  logTypeColors,
  LogTypeMetadata,
  ParentRelationshipMetadata,
  RevisionState,
  revisionStatecolors,
  RevisionStateMetadata,
  RevisionStateStyle,
} from 'src/app/zzz-generated';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { RendererConvertUtil } from './canvas/convertutil';

/**
 * ViewModel for revision legend item.
 */
interface RevisionLegendViewModel {
  label: string;
  icon: string;
  style: RevisionStateStyle;
  color: string;
}

/**
 * ViewModel for event legend item.
 */
interface EventLegendViewModel {
  label: string;
  color: string;
}

/**
 * ViewModel for timeline type legend item.
 */
interface TimelineTypeLegendViewModel {
  label: string;
  backgroundColor: string;
  color: string;
  hint: string;
}

/**
 * Component for displaying the legend of the timeline.
 * It shows the explanation of icons and colors used in the timeline.
 */
@Component({
  selector: 'khi-timeline-legend',
  templateUrl: './timeline-legend.component.html',
  styleUrls: ['./timeline-legend.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    MatButtonToggleModule,
    MatExpansionModule,
  ],
})
export class TimelineLegendComponent {
  readonly RevisionStateStyle = RevisionStateStyle;
  readonly TimelineLayer = TimelineLayer;

  /**
   * Whether the legend is expanded.
   */
  expanded = model(false);

  /**
   * The currently selected legend type ('revisions' or 'events').
   */
  legendType = model<string>('revisions');

  /**
   * The timeline data to generate legends for.
   */
  timeline = input<ResourceTimeline | null>(null);

  /**
   * Computed ViewModel for the timeline type legend.
   */
  timelineTypeLegend = computed<TimelineTypeLegendViewModel | null>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return null;
    }
    const metadata = ParentRelationshipMetadata[timeline.parentRelationship];

    return {
      label: metadata.label,
      color: RendererConvertUtil.hdrColorToCSSColor(metadata.color),
      backgroundColor: RendererConvertUtil.hdrColorToCSSColor(
        metadata.backgroundColor,
      ),
      hint: metadata.hint,
    };
  });

  /**
   * Computed list of ViewModels for revision legends found in the timeline.
   */
  revisionLegends = computed<RevisionLegendViewModel[]>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return [];
    }
    const revisionStates = new Set<RevisionState>();
    for (const revision of timeline.revisions) {
      revisionStates.add(revision.stateRaw);
    }
    return Array.from(revisionStates).map<RevisionLegendViewModel>((state) => {
      const md = RevisionStateMetadata[state];
      return {
        label: md.label,
        icon: md.icon,
        style: md.style,
        color: RendererConvertUtil.hdrColorToCSSColor(
          revisionStatecolors[md.cssSelector],
        ),
      };
    });
  });

  /**
   * Computed list of ViewModels for event legends found in the timeline.
   */
  eventLegends = computed<EventLegendViewModel[]>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return [];
    }
    const eventTypes = new Set<LogType>();
    for (const event of timeline.events) {
      eventTypes.add(event.logType);
    }
    return Array.from(eventTypes).map<EventLegendViewModel>((type) => {
      const md = LogTypeMetadata[type];
      return {
        label: md.label,
        color: RendererConvertUtil.hdrColorToCSSColor(logTypeColors[md.label]),
      };
    });
  });
}
