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

import { Component, computed, inject, input } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { combineLatest, switchMap } from 'rxjs';
import { WaypointManagerService } from '../waypoint-manager.service';
import { DiagramViewportService } from '../diagram-viewport.service';

/**
 * Defines the possible shapes for arrow heads and tails
 */
export enum ArrowShape {
  None = 'none',
  Circle = 'circle',
  Arrow = 'arrow',
}

/**
 * Defines the waypoint information for connecting arrows
 * Contains area ID and anchor position coordinates
 */
export interface WayPoint {
  areaID: string;
  anchorX: number;
  anchorY: number;
}

@Component({
  // eslint-disable-next-line @angular-eslint/component-selector
  selector: '[diagram-svg-arrow]',
  templateUrl: './diagram-svg-arrow.component.html',
})
export class DiagramSVGArrowComponent {
  /**
   * Reference to the ArrowShape enum for use in the template
   */
  ArrowShape = ArrowShape;

  private readonly waypointManager = inject(WaypointManagerService);

  private readonly viewportService = inject(DiagramViewportService);

  /**
   * Array of waypoints that define the path of the arrow
   * Required input that must be provided when using the component
   */
  readonly waypoints = input.required<WayPoint[]>();

  /**
   * The thickness (stroke width) of the arrow line
   * Default value is 1
   */
  readonly thickness = input<number>(1);

  /**
   * The shape to display at the head (start) of the arrow
   * Default is None (no shape)
   */
  readonly headShape = input<ArrowShape>(ArrowShape.None);

  /**
   * The size of the arrow head shape
   * Default value is 10
   */
  readonly headSize = input<number>(10);

  /**
   * Rotation angle in degrees for the arrow head
   * Default value is 0 (no rotation)
   */
  readonly headRotate = input<number>(0);

  /**
   * The shape to display at the tail (end) of the arrow
   * Default is None (no shape)
   */
  readonly tailShape = input<ArrowShape>(ArrowShape.None);

  /**
   * The size of the arrow tail shape
   * Default value is 10
   */
  readonly tailSize = input<number>(10);

  /**
   * Rotation angle in degrees for the arrow tail
   * Default value is 0 (no rotation)
   */
  readonly tailRotate = input<number>(0);

  /**
   * Signal tracking the actual coordinates of each waypoint
   * Converts the logical waypoints to actual DOM coordinates
   */
  readonly waypointCoordinates = toSignal(
    toObservable(this.waypoints).pipe(
      switchMap((waypoints) =>
        combineLatest(
          waypoints.map((p) =>
            this.waypointManager.monitorWaypoint(
              p.areaID,
              new DOMPoint(p.anchorX, p.anchorY),
            ),
          ),
        ),
      ),
    ),
  );

  /**
   * Signal tracking the current state of the viewport
   * Used to adjust arrow rendering based on viewport transformations
   */
  readonly viewportState = toSignal(this.viewportService.viewportState);

  /**
   * Computed signal that extracts the current viewport scale factor
   * Defaults to 1 if viewport state is not available
   */
  readonly viewportScale = computed(
    () => this.viewportState()?.contentScale ?? 1,
  );

  /**
   * Computed signal that generates the SVG polyline points string
   * Creates a space-separated list of x,y coordinates for the polyline
   */
  readonly polylinePoints = computed(() => {
    const waypoints = this.waypointCoordinates();
    if (!waypoints) {
      return '';
    }
    return waypoints.map((p) => `${p.x},${p.y}`).join(' ');
  });

  /**
   * Computed signal that determines the position of the arrow head
   * Returns the first waypoint's coordinates or default point if none exist
   */
  readonly headPosition = computed(() => {
    const waypoints = this.waypointCoordinates();
    if (!waypoints || waypoints.length === 0) {
      return new DOMPoint();
    }
    return waypoints[0];
  });

  /**
   * Computed signal that determines the position of the arrow tail
   * Returns the last waypoint's coordinates or default point if none exist
   */
  readonly tailPosition = computed(() => {
    const waypoints = this.waypointCoordinates();
    if (!waypoints || waypoints.length === 0) {
      return new DOMPoint();
    }
    return waypoints[waypoints.length - 1];
  });

  /**
   * Creates an SVG transform attribute string
   * @param x - X-coordinate for translation
   * @param y - Y-coordinate for translation
   * @param rotate - Rotation angle in degrees
   * @param scale - Scale factor
   * @returns SVG transform attribute string
   */
  createTransform(x: number, y: number, rotate: number, scale: number): string {
    return `translate(${x},${y}) rotate(${rotate}) scale(${scale})`;
  }
}
