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

import { CommonModule } from '@angular/common';
import { Component, input, output } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Represents a view model for a single resource reference link.
 */
export interface ResourceRefAnnotationViewModel {
  label: string;
  timelineId: number;
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
  imports: [CommonModule, MatIconModule, KHIIconRegistrationModule],
})
export class ResourceReferenceListComponent {
  /**
   * A list of resolved resource references to display.
   */
  refs = input<ResourceRefAnnotationViewModel[]>([]);

  /**
   * Input tracking the currently selected timeline to visually indicate selection state.
   */
  selectedTimeline = input<ReadonlyDomainElement<Timeline> | null>(null);

  /**
   * Output emitted when a timeline is clicked.
   */
  timelineSelected = output<number>();

  /**
   * Output emitted when a timeline is hovered.
   */
  timelineHighlighted = output<number>();

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
}
