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
  AfterViewInit,
  Component,
  computed,
  ElementRef,
  inject,
  OnDestroy,
  signal,
  viewChild,
  ViewContainerRef,
} from '@angular/core';
import { DiagramViewportService } from './diagram-viewport.service';
import { CommonModule } from '@angular/common';
import { fromEvent, Subject, takeUntil } from 'rxjs';

const scalingStep = 0.03;

/**
 * Viewport for the diagram view.
 * It supports XY scrolling and reports visible area info and content size info to the DiagramViewportService.
 */
@Component({
  selector: 'diagram-viewport',
  templateUrl: './diagram-viewport.component.html',
  styleUrls: ['./diagram-viewport.component.sass'],
  imports: [CommonModule],
  host: {
    '(scroll)': 'onScroll()',
    '(wheel)': 'onWheel($event)',
  },
})
export class DiagramViewportComponent implements AfterViewInit, OnDestroy {
  private readonly destroy = new Subject();
  private readonly contentHostElement =
    viewChild<ElementRef<HTMLDivElement>>('contentHost');
  private readonly viewportService = inject(DiagramViewportService);

  readonly scalingFactor = signal(1);

  readonly styleTransform = computed(() => `scale(${this.scalingFactor()})`);

  constructor(private readonly hostElement: ViewContainerRef) {}

  ngAfterViewInit(): void {
    this.reportVisibleArea();

    // monitor window resize event to recalculate the viewport size.
    fromEvent(window, 'resize')
      .pipe(takeUntil(this.destroy))
      .subscribe(() => {
        this.reportVisibleArea();
      });
  }

  onScroll() {
    this.reportVisibleArea();
  }

  onWheel(event: WheelEvent) {
    if (event.shiftKey) {
      const viewportNativeElement = this.hostElement.element
        .nativeElement as HTMLElement;
      const viewportRect = viewportNativeElement.getBoundingClientRect();
      const contentHostElement = this.contentHostElement();
      if (!contentHostElement) {
        return;
      }
      const contentHostRect =
        contentHostElement.nativeElement.getBoundingClientRect();

      const s = this.scalingFactor();
      const minScale = Math.min(
        viewportRect.width / contentHostRect.width,
        viewportRect.height / contentHostRect.height,
      );
      const sn = Math.max(minScale, s + event.deltaY * -1 * scalingStep);
      if (sn === s) return;

      this.scalingFactor.set(sn);
      viewportNativeElement.scrollLeft = calculateNewScrollAmount(
        event.clientX,
        viewportNativeElement.scrollLeft,
        viewportRect.left,
        s,
        sn,
      );
      viewportNativeElement.scrollTop = calculateNewScrollAmount(
        event.clientY,
        viewportNativeElement.scrollTop,
        viewportRect.top,
        s,
        sn,
      );

      event.preventDefault();
    }
  }

  private reportVisibleArea() {
    const contentHost = this.contentHostElement();
    const scale = this.scalingFactor();
    if (contentHost) {
      const contentHostRect = contentHost.nativeElement.getBoundingClientRect();
      const viewportElement = this.hostElement.element
        .nativeElement as HTMLElement;
      const viewportRect = viewportElement.getBoundingClientRect();
      const scrollY = viewportElement.scrollTop;
      const scrollX = viewportElement.scrollLeft;
      this.viewportService.notifyViewportChange(
        contentHostRect.width / scale,
        contentHostRect.height / scale,
        scale,
        scrollX / contentHostRect.width,
        scrollY / contentHostRect.height,
        Math.min(
          1 - scrollX / contentHostRect.width,
          viewportRect.width / contentHostRect.width,
        ),
        Math.min(
          1 - scrollY / contentHostRect.height,
          viewportRect.height / contentHostRect.height,
        ),
      );
    }
  }

  ngOnDestroy(): void {
    this.destroy.next(undefined);
    this.destroy.complete();
  }
}

/**
 * Calculates new scroll amount which doesn't change the scaled coordinate in content host space.
 * This is needed to scale in/out around the mouse pointer.
 * @returns calculated newer scroll amount
 */
export function calculateNewScrollAmount(
  client: number,
  oldScroll: number,
  viewport: number,
  oldScale: number,
  newScale: number,
): number {
  // Viewport bounding rect X: VPx
  // Viewport scrolling X(current): SCx
  // Viewport scrolling X(new): SNx
  // Mouse event client X: MCx
  // Current scale: S
  // New scale: Sn
  // (MCx - VPx + SCx)/S = (MCx - VPx + SNx)/Sn
  // { (MCx - VPx + SCx) : mouse pointer location in content host space }
  // The equation represents mouse position in content host space keeping its position with the new scale and new calculated scroll amount.
  // (MCx - VPx + SCx) * Sn/S = MCx - VPx + SNx
  // SNx = VPx - MCx + (MCx - VPx +SCx) * Sn / S
  return (
    viewport - client + ((client - viewport + oldScroll) * newScale) / oldScale
  );
}
