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
  Observable,
  ReplaySubject,
  shareReplay,
  Subject,
} from 'rxjs';
import { LOD } from './lod.service';

/**
 * A data type representing the viewport area status.
 */
export interface DiagramViewportState {
  contentWidth: number;
  contentHeight: number;
  contentScale: number;

  /**
   * Each viewport related fields are normalized to [0,1]
   */
  viewportLeft: number;
  viewportTop: number;
  viewportWidth: number;
  viewportHeight: number;
}

/**
 * Providing XY-direction virtual scroll features.
 * This class computes the container sizes and compute child element LODs(Level of Details).
 */
export class DiagramViewportService {
  private readonly viewportStateSubject =
    new ReplaySubject<DiagramViewportState>(1);

  /**
   * The size and location of entire viewport.
   */
  readonly viewportState = this.viewportStateSubject.pipe(
    distinctUntilChanged(
      (a, b) =>
        a.contentWidth === b.contentWidth &&
        a.contentHeight === b.contentHeight &&
        a.contentScale === b.contentScale &&
        a.viewportLeft === b.viewportLeft &&
        a.viewportTop === b.viewportTop &&
        a.viewportWidth === b.viewportWidth &&
        a.viewportHeight === b.viewportHeight,
    ),
    shareReplay(1),
  );

  private lodSubjects: { [id: string]: Subject<LOD> } = {};

  /**
   * Notifies viewport state changes to subscribers.
   */
  public notifyViewportChange(
    contentWidth: number,
    contentHeight: number,
    contentScale: number,
    viewportLeft: number,
    viewportTop: number,
    viewportWidth: number,
    viewportHeight: number,
  ) {
    this.viewportStateSubject.next({
      contentWidth,
      contentHeight,
      contentScale,
      viewportLeft,
      viewportTop,
      viewportWidth,
      viewportHeight,
    });
  }

  /**
   * Monitor the LOD of the specified element.
   */
  public monitorLOD(id: string): Observable<LOD> {
    if (!this.lodSubjects[id]) {
      this.lodSubjects[id] = new ReplaySubject<LOD>(1);
    }
    return this.lodSubjects[id].pipe(distinctUntilChanged());
  }

  /**
   * Notify changed LOD of the specified element.
   */
  public notifyLOD(id: string, lod: LOD) {
    if (!this.lodSubjects[id]) {
      this.lodSubjects[id] = new ReplaySubject<LOD>(1);
    }
    this.lodSubjects[id].next(lod);
  }

  /**
   * Remove specified diagram element.
   */
  public removeDiagramElement(id: string) {
    delete this.lodSubjects[id];
  }
}
