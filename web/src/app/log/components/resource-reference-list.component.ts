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
import { Component, input, output, signal } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { OverlayModule, ConnectionPositionPair } from '@angular/cdk/overlay';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineType } from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  ResourceHierarchyOverlayComponent,
  ResourcePathNodeViewModel,
  getTimelineStyle,
} from 'src/app/log/components/resource-hierarchy-overlay.component';

/**
 * Represents a view model for a single resource reference link.
 */
export interface ResourceRefAnnotationViewModel {
  readonly label: string;
  readonly timelineId: number;
  readonly name: string;
  readonly type: TimelineType;
  readonly pathNodes: ResourcePathNodeViewModel[];
}

/**
 * `ResourceReferenceListComponent` renders a list of related resources extracted from a loaded log.
 * It displays clickable chips that allow the user to highlight or select specific timelines
 * directly from the log details view.
 */
@Component({
  selector: 'khi-resource-reference-list',
  standalone: true,
  templateUrl: './resource-reference-list.component.html',
  styleUrl: './resource-reference-list.component.scss',
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    OverlayModule,
    ResourceHierarchyOverlayComponent,
  ],
})
export class ResourceReferenceListComponent {
  /**
   * A list of resolved resource references to display.
   */
  readonly refs = input<ResourceRefAnnotationViewModel[]>([]);

  /**
   * Input tracking the currently selected timeline to visually indicate selection state.
   */
  readonly selectedTimeline = input<ReadonlyDomainElement<Timeline> | null>(
    null,
  );

  /**
   * Output emitted when a timeline is clicked.
   */
  readonly timelineSelected = output<number>();

  /**
   * Output emitted when a timeline is hovered.
   */
  readonly timelineHighlighted = output<number>();

  /**
   * Signal tracking the currently active overlay timeline ID.
   */
  readonly activeOverlayTimelineId = signal<number | null>(null);

  /**
   * Connection position pairs for the hierarchy overlay popup.
   */
  readonly overlayPositions: ConnectionPositionPair[] = [
    new ConnectionPositionPair(
      { originX: 'start', originY: 'bottom' },
      { overlayX: 'start', overlayY: 'top' },
      0,
      4,
    ),
    new ConnectionPositionPair(
      { originX: 'start', originY: 'top' },
      { overlayX: 'start', overlayY: 'bottom' },
      0,
      -4,
    ),
  ];

  /**
   * Select the timeline by its ID.
   */
  public selectResource(timelineId: number) {
    this.timelineSelected.emit(timelineId);
  }

  /**
   * Highlight the timeline by its ID.
   */
  public highlightResource(timelineId: number) {
    this.timelineHighlighted.emit(timelineId);
  }

  /**
   * Converts a TimelineType into CSS custom property styles for background and chip colors.
   */
  public getTimelineStyle(type: TimelineType): Record<string, string> {
    return getTimelineStyle(type);
  }
}
