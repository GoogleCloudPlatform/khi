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
  Directive,
  ElementRef,
  inject,
  input,
  OnDestroy,
} from '@angular/core';
import { WaypointManagerService } from './waypoint-manager.service';
import {
  animationFrames,
  distinctUntilChanged,
  map,
  shareReplay,
  Subject,
  takeUntil,
} from 'rxjs';
import {
  DIAGRAM_ELEMENT_ROLE,
  DiagramElementRole,
} from './diagram-element.directive';

/**
 * DiagramWaypointAreaDirective registers its element positions on WaypointManagerService to use these positions to draw connected arrows.
 */
@Directive({
  selector: '[diagramWaypointArea]',
})
export class DiagramWaypointAreaDirective implements AfterViewInit, OnDestroy {
  private readonly destroyed = new Subject();
  private readonly element = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly role = inject(DIAGRAM_ELEMENT_ROLE);
  private readonly waypointManager = inject(WaypointManagerService);

  /**
   * The identifier of waypoint area.
   */
  readonly waypointAreaID = input.required<string>();

  /**
   * Observable that tracks the bounding rectangle of the waypoint area
   * Updates on animation frames and only emits when dimensions or position change
   */
  readonly waypointAreaRectObservable = animationFrames().pipe(
    takeUntil(this.destroyed),
    map(() => this.element.nativeElement.getBoundingClientRect()),
    distinctUntilChanged(
      (a, b) =>
        a.x === b.x &&
        a.y === b.y &&
        a.width === b.width &&
        a.height === b.height,
    ),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  /**
   * Lifecycle hook that registers this waypoint area with the manager
   * Only registers if the element has CONTENT role (not to capture element locations in minimap)
   */
  ngAfterViewInit(): void {
    if (this.role === DiagramElementRole.Content) {
      this.waypointManager.registerWaypointArea(this.waypointAreaID(), this);
    }
  }

  /**
   * Lifecycle hook that unregisters this waypoint area and cleans up resources
   * Only performs cleanup if the element has CONTENT role (not to capture element locations in minimap)
   */
  ngOnDestroy(): void {
    if (this.role === DiagramElementRole.Content) {
      this.destroyed.next(undefined);
      this.waypointManager.unregisterWaypointArea(this.waypointAreaID());
    }
  }
}
