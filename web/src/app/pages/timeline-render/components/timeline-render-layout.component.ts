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
  ElementRef,
  input,
  model,
  output,
  viewChild,
} from '@angular/core';
import { TimelineEvidenceDeckComponent } from 'src/app/pages/timeline-render/components/timeline-evidence-deck.component';
import { TimelineSceneOverlayComponent } from 'src/app/pages/timeline-render/components/timeline-scene-overlay.component';
import {
  TimelineSceneViewModel,
  TimelineSessionJumpTarget,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
import { TIMELINE_RENDER_HORIZONTAL_OVERDRAW_PX } from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { Log } from 'src/app/store/domain/log';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  TimelineChartStyle,
  TimelineRulerStyle,
} from 'src/app/timeline/components/style-model';
import { TimelineFrameComponent } from 'src/app/timeline/components/timeline-frame.component';

/**
 * Layout dumb component combining the top marker band, timeline frame, vector overlay, and bottom evidence deck.
 */
@Component({
  selector: 'khi-timeline-render-layout',
  templateUrl: './timeline-render-layout.component.html',
  styleUrls: ['./timeline-render-layout.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    TimelineFrameComponent,
    TimelineSceneOverlayComponent,
    TimelineEvidenceDeckComponent,
  ],
})
export class TimelineRenderLayoutComponent {
  private readonly timelineFrame = viewChild<TimelineFrameComponent>(
    TimelineFrameComponent,
  );
  private readonly deckContainer =
    viewChild<ElementRef<HTMLElement>>('deckContainer');

  /**
   * Unique identifier of the currently loaded InspectionData instance.
   */
  readonly inspectionDataUniqueID = input.required<string>();

  /**
   * Style repository supplying timeline row heights and colors.
   */
  readonly styleStore = input.required<StyleStoreLike>();

  /**
   * Visual chart style configuration for WebGL rendering.
   */
  readonly chartStyle = input.required<TimelineChartStyle>();

  /**
   * Visual ruler style configuration for header ticks.
   */
  readonly rulerStyle = input.required<TimelineRulerStyle>();

  /**
   * Ordered subset of timelines rendered in the frame.
   */
  readonly timelines =
    input.required<readonly ReadonlyDomainElement<Timeline>[]>();

  /**
   * Set of collapsed timeline IDs.
   */
  readonly collapsedTimelineIds = input.required<ReadonlySet<number>>();

  /**
   * Timezone offset in hours applied to ruler labels.
   */
  readonly timezoneShiftHours = input.required<number>();

  /**
   * Horizontal zoom scale in pixels per nanosecond.
   */
  readonly pixelsPerNs = model.required<number>();

  /**
   * Leftmost visible timestamp of the chart viewport in nanoseconds.
   */
  readonly viewportLeftTimeNs = model.required<bigint>();

  /**
   * Minimum query log timestamp in nanoseconds.
   */
  readonly minQueryLogTimeNs = input.required<bigint>();

  /**
   * Maximum query log timestamp in nanoseconds.
   */
  readonly maxQueryLogTimeNs = input.required<bigint>();

  /**
   * Complete list of log domain elements for histogram rendering.
   */
  readonly allLogs = input.required<ReadonlyDomainElement<Log>[]>();

  /**
   * Bitset of log IDs matching the active filter.
   */
  readonly filteredLogIds = input.required<IdBitset>();

  /**
   * Precomputed scene geometry and annotation ViewModel.
   */
  readonly scene = input.required<TimelineSceneViewModel>();

  /**
   * Enables interactive toolbar, scrollbars, and session jump links when opened in a browser.
   */
  readonly interactiveMode = input<boolean>(false);

  /**
   * Active scene ID displayed in interactive mode.
   */
  readonly sceneId = input<string>('');

  /**
   * Optional parent scene ID displayed in interactive mode.
   */
  readonly parentSceneId = input<string>('');

  /**
   * Active inspection ID displayed in interactive mode.
   */
  readonly inspectionId = input<string>('');

  /**
   * Optional error banner message when loading an interactive scene fails.
   */
  readonly errorMessage = input<string>('');

  /**
   * Synchronized badge hover state between timeline pins and deck cards.
   */
  readonly hoveredBadge = input<string | null>(null);

  /**
   * Horizontal canvas overdraw in CSS pixels used for seamless pan/zoom.
   */
  readonly horizontalOverdrawInPx = input<number>(
    TIMELINE_RENDER_HORIZONTAL_OVERDRAW_PX,
  );

  /**
   * Emits hovered badge changes from overlay pins or deck cards.
   */
  readonly badgeHover = output<string | null>();

  /**
   * Emits jump navigation into a full KHI session for a specific revision or log.
   */
  readonly jumpToSession = output<TimelineSessionJumpTarget>();

  /**
   * Emits when the user clicks Open in Full KHI in the interactive toolbar.
   */
  readonly openFullSession = output<void>();

  /**
   * Reports whether the underlying TimelineFrameComponent is ready to render frames.
   */
  public isFrameReadyToRender(): boolean {
    const frame = this.timelineFrame();
    return frame ? frame.readyToRender() : false;
  }

  /**
   * Synchronizes viewport state and triggers canvas redraw on the underlying TimelineFrameComponent.
   */
  public syncViewportAndInvalidate(
    viewportLeftTimeNs: bigint,
    pixelsPerNs: number,
  ): void {
    this.timelineFrame()?.syncViewportAndInvalidate(
      viewportLeftTimeNs,
      pixelsPerNs,
    );
  }

  /**
   * Measures the rendered DOM height of the bottom evidence deck container in pixels.
   */
  public measureDeckHeightPx(): number {
    const el = this.deckContainer()?.nativeElement;
    if (!el) {
      return 0;
    }
    return Math.ceil(el.getBoundingClientRect().height);
  }

  /**
   * Relays badge hover updates to the smart component.
   */
  onBadgeHover(badge: string | null): void {
    this.badgeHover.emit(badge);
  }

  /**
   * Relays targeted session jump navigation to the smart component.
   */
  onJumpToSession(target: TimelineSessionJumpTarget): void {
    this.jumpToSession.emit(target);
  }

  /**
   * Relays full KHI session open clicks from the interactive toolbar.
   */
  onOpenFullSessionClick(): void {
    this.openFullSession.emit();
  }
}
