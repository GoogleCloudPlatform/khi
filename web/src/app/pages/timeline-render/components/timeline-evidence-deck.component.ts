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
  TimelineExcerptKind,
  TimelineExcerptLineKind,
} from 'src/app/common/schema/timeline-render-types';
import {
  ExcerptDeckViewModel,
  TimelineSessionJumpTarget,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';

/**
 * Dumb component rendering the 32px S-curve SVG tether strip and bottom evidence deck of diff and log cards.
 */
@Component({
  selector: 'khi-timeline-evidence-deck',
  templateUrl: './timeline-evidence-deck.component.html',
  styleUrls: ['./timeline-evidence-deck.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TimelineEvidenceDeckComponent {
  /**
   * Exports TimelineExcerptKind enum to the template.
   */
  protected readonly TimelineExcerptKind = TimelineExcerptKind;

  /**
   * Exports TimelineExcerptLineKind enum to the template.
   */
  protected readonly TimelineExcerptLineKind = TimelineExcerptLineKind;

  /**
   * Precomputed evidence deck cards and S-curve tether geometry.
   */
  readonly deck = input.required<ExcerptDeckViewModel>();

  /**
   * Total horizontal stage width in pixels matching the timeline canvas above.
   */
  readonly stageWidthPx = input.required<number>();

  /**
   * Shows interactive Open Diff and Open Log jump buttons when true.
   */
  readonly interactiveMode = input<boolean>(false);

  /**
   * Currently hovered badge identifier synchronized with timeline pins.
   */
  readonly hoveredBadge = input<string | null>(null);

  /**
   * Emits the hovered badge identifier or null when pointer leaves a card.
   */
  readonly badgeHover = output<string | null>();

  /**
   * Emits a target resource revision or log entry when an interactive jump button is clicked.
   */
  readonly jumpToSession = output<TimelineSessionJumpTarget>();

  /**
   * Highlights the matching timeline pin and tether when hovering over a card.
   */
  onCardMouseEnter(badge: string): void {
    if (!this.interactiveMode()) {
      return;
    }
    this.badgeHover.emit(badge);
  }

  /**
   * Clears badge highlight when pointer leaves a card.
   */
  onCardMouseLeave(): void {
    if (!this.interactiveMode()) {
      return;
    }
    this.badgeHover.emit(null);
  }

  /**
   * Emits jump navigation into a full KHI session for the selected card.
   */
  onOpenClick(target: TimelineSessionJumpTarget): void {
    if (!this.interactiveMode()) {
      return;
    }
    this.jumpToSession.emit(target);
  }
}
