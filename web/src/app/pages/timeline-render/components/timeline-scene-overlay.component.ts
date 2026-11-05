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

import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
} from '@angular/core';
import {
  TimelineConnectorStyle,
  TimelineMarkerStyle,
} from 'src/app/common/schema/timeline-render-types';
import {
  TimelineSceneViewModel,
  TimelineSessionJumpTarget,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';

/**
 * Dumb overlay component rendering V1-V6 geometric annotations and V7/V8 pin badges over the timeline stage.
 */
@Component({
  selector: 'khi-timeline-scene-overlay',
  templateUrl: './timeline-scene-overlay.component.html',
  styleUrls: ['./timeline-scene-overlay.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TimelineSceneOverlayComponent {
  /**
   * Exports TimelineMarkerStyle enum to the template.
   */
  protected readonly TimelineMarkerStyle = TimelineMarkerStyle;

  /**
   * Exports TimelineConnectorStyle enum to the template.
   */
  protected readonly TimelineConnectorStyle = TimelineConnectorStyle;

  /**
   * Precomputed scene geometry and annotation items.
   */
  readonly scene = input.required<TimelineSceneViewModel>();

  /**
   * Enables pointer interaction for hover synchronization and session jump clicks.
   */
  readonly interactiveMode = input<boolean>(false);

  /**
   * Currently hovered badge identifier synchronized between timeline pins and deck cards.
   */
  readonly hoveredBadge = input<string | null>(null);

  /**
   * Emits the hovered badge identifier or null when pointer leaves.
   */
  readonly badgeHover = output<string | null>();

  /**
   * Emits a target resource revision or log entry when an interactive badge or pill is clicked.
   */
  readonly jumpToSession = output<TimelineSessionJumpTarget>();

  /**
   * Notifies parent when pointer enters a badged pin or marker pill.
   */
  onBadgeMouseEnter(badge: string): void {
    if (!this.interactiveMode()) {
      return;
    }
    this.badgeHover.emit(badge);
  }

  /**
   * Clears badge hover highlight when pointer leaves.
   */
  onBadgeMouseLeave(): void {
    if (!this.interactiveMode()) {
      return;
    }
    this.badgeHover.emit(null);
  }

  /**
   * Emits jump navigation when an interactive annotation with a session target is clicked.
   */
  onJumpClick(target: TimelineSessionJumpTarget | null): void {
    if (!this.interactiveMode() || !target) {
      return;
    }
    this.jumpToSession.emit(target);
  }
}
