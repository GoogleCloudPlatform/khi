/**
 * Copyright 2025 Google LLC
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
  Component,
  computed,
  ElementRef,
  inject,
  viewChild,
} from '@angular/core';
import { DiagramViewportService } from './diagram-viewport.service';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import {
  animationFrames,
  distinctUntilChanged,
  filter,
  map,
  switchMap,
} from 'rxjs';

/**
 * Overlay SVG area put on the diagram viewport.
 * This is used for drawing arrows because it would be performant and easier to draw many arrows to connect elements.
 * SVG coordinate space is screen viewport space in px.
 */
@Component({
  selector: 'diagram-svg-viewport-overlay',
  templateUrl: './diagram-svg-viewport-overlay.component.html',
  styleUrl: './diagram-svg-viewport-overlay.component.sass',
})
export class DiagramSVGViewportOverlayComponent {
  private readonly svgElement = viewChild<ElementRef<SVGElement>>('svgElement');
  private readonly viewportService = inject(DiagramViewportService);

  /**
   * Signal tracking the current state of the viewport
   * Contains dimensions, scale, and position information with default initial values
   */
  viewportState = toSignal(this.viewportService.viewportState, {
    initialValue: {
      contentWidth: 1,
      contentHeight: 1,
      contentScale: 1,
      viewportLeft: 0,
      viewportTop: 0,
      viewportWidth: 1,
      viewportHeight: 1,
    },
  });

  /**
   * Computed signal that converts the viewport state to a DOMRect
   * Creates a rectangle representing the visible viewport area in content coordinates
   */
  viewportRect = computed(() => {
    const state = this.viewportState();
    return new DOMRect(
      state.viewportLeft * state.contentWidth * state.contentScale,
      state.viewportTop * state.contentHeight * state.contentScale,
      state.viewportWidth * state.contentWidth * state.contentScale,
      state.viewportHeight * state.contentHeight * state.contentScale,
    );
  });

  /**
   * Signal tracking the SVG element's bounding client rectangle
   * Updates on animation frames when the element's size or position changes
   */
  svgRect = toSignal(
    toObservable(this.svgElement).pipe(
      switchMap((svgElement) =>
        animationFrames().pipe(
          filter(() => svgElement !== undefined),
          map(() => svgElement!.nativeElement.getBoundingClientRect()),
          distinctUntilChanged(
            (a, b) =>
              a.x === b.x &&
              a.y === b.y &&
              a.width === b.width &&
              a.height === b.height,
          ),
        ),
      ),
    ),
  );

  /**
   * Computed signal that generates the SVG viewBox attribute string
   * Creates a space-separated string of x, y, width, and height values
   */
  viewBox = computed(() => {
    const svgRect = this.svgRect();
    if (!svgRect) {
      return '0 0 0 0';
    }
    return `${svgRect.x} ${svgRect.y} ${svgRect.width} ${svgRect.height}`;
  });
}
