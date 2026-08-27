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

import { Component, computed, input, model, output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSliderModule } from '@angular/material/slider';
import { MatTooltip } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { DEFAULT_DELETION_THRESHOLD_SECONDS } from 'src/app/common/schema/graph-schema';
import { formatDurationSeconds } from 'src/app/utils/time-format-util';

/**
 * Defines the minimum retention threshold in seconds for the toolbar slider.
 */
export const MIN_DELETION_THRESHOLD_SECONDS = 10;

/**
 * Defines the maximum retention threshold in seconds for the toolbar slider.
 */
export const MAX_DELETION_THRESHOLD_SECONDS = 3600;

/**
 * Defines the step interval in seconds for the toolbar slider.
 */
export const STEP_DELETION_THRESHOLD_SECONDS = 10;

/**
 * Renders the toolbar controls for the architecture graph view.
 * Provides a deletion duration slider, a fit to view button, and graph download actions.
 */
@Component({
  selector: 'khi-graph-toolbar',
  templateUrl: './graph-toolbar.component.html',
  styleUrls: ['./graph-toolbar.component.scss'],
  imports: [
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatSliderModule,
    MatTooltip,
    KHIIconRegistrationModule,
  ],
})
export class GraphToolbarComponent {
  /**
   * Defines the minimum deletion retention threshold in seconds for the slider.
   */
  protected readonly minDeletionThresholdSeconds =
    MIN_DELETION_THRESHOLD_SECONDS;

  /**
   * Defines the maximum deletion retention threshold in seconds for the slider.
   */
  protected readonly maxDeletionThresholdSeconds =
    MAX_DELETION_THRESHOLD_SECONDS;

  /**
   * Defines the step interval in seconds for the deletion retention slider.
   */
  protected readonly stepDeletionThresholdSeconds =
    STEP_DELETION_THRESHOLD_SECONDS;

  /**
   * Holds the deletion retention threshold in seconds for two-way binding.
   */
  readonly deletionThresholdSeconds = model<number>(
    DEFAULT_DELETION_THRESHOLD_SECONDS,
  );

  /**
   * Indicates whether the graph is currently loading.
   */
  readonly isLoading = input<boolean>(false);

  /**
   * Emits when the user requests fitting the graph to the view bounds.
   */
  readonly fitToView = output<void>();

  /**
   * Emits when the user requests downloading the graph as an SVG image.
   */
  readonly downloadSvg = output<void>();

  /**
   * Emits when the user requests downloading the graph as a PNG image.
   */
  readonly downloadPng = output<void>();

  /**
   * Computes a human-readable display string representing the deletion threshold.
   */
  protected readonly formattedDeletionThreshold = computed<string>(() =>
    formatDurationSeconds(this.deletionThresholdSeconds()),
  );
}
