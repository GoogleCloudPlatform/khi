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
  distinctUntilChanged,
  map,
  Observable,
  ReplaySubject,
  Subject,
} from 'rxjs';
import { DiagramWaypointAreaDirective } from './waypoint-area.directive';

/**
 * WaypointArea represents rectangular area on the diagram where can be used as a connector of lines.
 */

export interface WaypointAreaViewportSpace {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface OptionalPosition {
  x?: number;
  y?: number;
}

/**
 * WaypointManagerService manages the points on diagrams.
 * Arrows or lines uses the named points to draw.
 */
export class WaypointManagerService {
  private readonly waypointAreaSubjects: { [id: string]: Subject<DOMRect> } =
    {};

  /**
   * monitorWaypoint get the observable to monitor a point at the specified area.
   * positionInArea is [0,1] normalized DOMPoint locates the relative location in the area. [0,0] means top left, [0,1] means bottom left for example.
   * monitorWaypoint returns undefined for a dimention when the dimention in the given positionInArea is undefined. The coordinate would be determined regarding the other positions.
   */
  monitorWaypoint(
    areaID: string,
    positionInArea: OptionalPosition,
  ): Observable<OptionalPosition> {
    if (this.waypointAreaSubjects[areaID] === undefined) {
      this.waypointAreaSubjects[areaID] = new ReplaySubject<DOMRect>(1);
    }
    return this.waypointAreaSubjects[areaID].pipe(
      map(
        (rect) =>
          ({
            x:
              positionInArea.x !== undefined
                ? rect.x + positionInArea.x * rect.width
                : undefined,
            y:
              positionInArea.y !== undefined
                ? rect.y + positionInArea.y * rect.height
                : undefined,
          }) as OptionalPosition,
      ),
      distinctUntilChanged((a, b) => a.x === b.x && a.y === b.y),
    );
  }

  /**
   * Registers a waypoint area with the specified ID
   * Associates the area with a directive and subscribes to its rect observable
   * @param areaID - Unique identifier for the waypoint area
   * @param area - The directive instance representing the area
   */
  registerWaypointArea(areaID: string, area: DiagramWaypointAreaDirective) {
    if (this.waypointAreaSubjects[areaID] === undefined) {
      this.waypointAreaSubjects[areaID] = new ReplaySubject<DOMRect>();
    }
    area.waypointAreaRectObservable.subscribe(
      this.waypointAreaSubjects[areaID],
    );
  }

  /**
   * Unregisters a waypoint area with the specified ID
   * Completes the associated subject and removes it from tracking
   * @param areaID - Unique identifier of the waypoint area to unregister
   */
  unregisterWaypointArea(areaID: string) {
    if (this.waypointAreaSubjects[areaID] !== undefined) {
      this.waypointAreaSubjects[areaID].complete();
      delete this.waypointAreaSubjects[areaID];
    }
  }
}
