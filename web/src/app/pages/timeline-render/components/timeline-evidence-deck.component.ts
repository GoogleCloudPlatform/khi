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
  afterEveryRender,
  computed,
  effect,
  input,
  output,
  signal,
  viewChild,
  viewChildren,
} from '@angular/core';
import {
  TimelineExcerptKind,
  TimelineExcerptLineKind,
} from 'src/app/common/schema/timeline-render-types';
import { buildExcerptTetherSvgPath } from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import {
  ExcerptCardAnchorPoint,
  ExcerptDeckViewModel,
  ExcerptTetherViewModel,
  TimelineSessionJumpTarget,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';

interface MeasuredDeckLayout {
  readonly layoutKey: string;
  readonly heightPx: number;
  readonly anchorsByCardId: ReadonlyMap<string, ExcerptCardAnchorPoint>;
}

/**
 * Dumb component rendering the background S-curve SVG tether layer and bottom evidence deck of diff and log cards.
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

  private readonly deckRoot = viewChild<ElementRef<HTMLElement>>('deckRoot');
  private readonly cardsGrid = viewChild<ElementRef<HTMLElement>>('cardsGrid');
  private readonly cardElements =
    viewChildren<ElementRef<HTMLElement>>('cardElement');

  private readonly measuredLayout = signal<MeasuredDeckLayout | null>(null);

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

  private readonly currentLayoutKey = computed<string>(() => {
    const deckVm = this.deck();
    const cardIds = deckVm.cards.map((c) => c.id).join(',');
    return `${this.stageWidthPx()}:${deckVm.gridPreset}:${cardIds}`;
  });

  /**
   * Computes total SVG height in pixels using DOM measurement when available or the estimated height fallback.
   */
  protected readonly resolvedSvgHeightPx = computed<number>(() => {
    const deckVm = this.deck();
    const measured = this.measuredLayout();
    if (measured && measured.layoutKey === this.currentLayoutKey()) {
      return measured.heightPx;
    }
    return deckVm.estimatedHeightPx;
  });

  /**
   * Resolves tether paths and joint positions aligned to actual rendered card top borders when measured in the DOM.
   */
  protected readonly resolvedTethers = computed<
    readonly ExcerptTetherViewModel[]
  >(() => {
    const deckVm = this.deck();
    const measured = this.measuredLayout();
    if (!measured || measured.layoutKey !== this.currentLayoutKey()) {
      return deckVm.tethers;
    }
    return deckVm.tethers.map((tether) => {
      const anchor = measured.anchorsByCardId.get(tether.cardId);
      if (!anchor) {
        return tether;
      }
      if (
        anchor.anchorX === tether.cardAnchorX &&
        anchor.anchorY === tether.cardAnchorY
      ) {
        return tether;
      }
      return {
        ...tether,
        cardAnchorX: anchor.anchorX,
        cardAnchorY: anchor.anchorY,
        jointCenterY: Math.round((anchor.anchorY - 2) * 10) / 10,
        svgPath: buildExcerptTetherSvgPath(
          tether.pinX,
          anchor.anchorX,
          anchor.anchorY,
        ),
      };
    });
  });

  constructor() {
    afterEveryRender(() => {
      this.syncMeasuredAnchors();
    });

    effect((onCleanup) => {
      const gridEl = this.cardsGrid()?.nativeElement;
      const cardRefs = this.cardElements();
      if (!gridEl || typeof ResizeObserver === 'undefined') {
        return;
      }
      const observer = new ResizeObserver(() => {
        this.syncMeasuredAnchors();
      });
      observer.observe(gridEl);
      for (const ref of cardRefs) {
        observer.observe(ref.nativeElement);
      }
      onCleanup(() => {
        observer.disconnect();
      });
    });
  }

  /**
   * Reads rendered card offsets from the DOM so multi-row tethers terminate accurately at each card's top border.
   */
  public syncMeasuredAnchors(): void {
    const rootEl = this.deckRoot()?.nativeElement;
    const gridEl = this.cardsGrid()?.nativeElement;
    const cardRefs = this.cardElements();
    if (!rootEl || !gridEl || cardRefs.length === 0) {
      if (this.measuredLayout() !== null) {
        this.measuredLayout.set(null);
      }
      return;
    }

    const rootHeightPx = rootEl.clientHeight;
    if (rootHeightPx <= 0) {
      return;
    }

    const gridLeftPx = gridEl.offsetLeft;
    const gridTopPx = gridEl.offsetTop;
    const nextAnchors = new Map<string, ExcerptCardAnchorPoint>();
    for (const ref of cardRefs) {
      const el = ref.nativeElement;
      const cardId = el.dataset['cardId'];
      if (!cardId || el.offsetWidth <= 0) {
        continue;
      }
      const anchorX =
        Math.round((gridLeftPx + el.offsetLeft + el.offsetWidth / 2) * 10) / 10;
      const anchorY = Math.round((gridTopPx + el.offsetTop) * 10) / 10;
      nextAnchors.set(cardId, { anchorX, anchorY });
    }
    if (nextAnchors.size === 0) {
      return;
    }

    const layoutKey = this.currentLayoutKey();
    const prev = this.measuredLayout();
    if (
      prev &&
      prev.layoutKey === layoutKey &&
      prev.heightPx === rootHeightPx &&
      areCardAnchorMapsEqual(prev.anchorsByCardId, nextAnchors)
    ) {
      return;
    }

    this.measuredLayout.set({
      layoutKey,
      heightPx: rootHeightPx,
      anchorsByCardId: nextAnchors,
    });
  }

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

function areCardAnchorMapsEqual(
  a: ReadonlyMap<string, ExcerptCardAnchorPoint>,
  b: ReadonlyMap<string, ExcerptCardAnchorPoint>,
): boolean {
  if (a.size !== b.size) {
    return false;
  }
  for (const [key, valA] of a) {
    const valB = b.get(key);
    if (
      !valB ||
      valA.anchorX !== valB.anchorX ||
      valA.anchorY !== valB.anchorY
    ) {
      return false;
    }
  }
  return true;
}
