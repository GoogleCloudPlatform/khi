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

import { inject, Injectable } from '@angular/core';
import {
  combineLatest,
  map,
  Observable,
  ReplaySubject,
  Subject,
  switchMap,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import { DiagramModel } from './diagram-model-types';
import { WindowConnectorService } from '../frame-connection/window-connector.service';
import {
  QUERY_CURRENT_INSPECTION_METADATA,
  QUERY_DIAGRAM_DATA,
} from 'src/app/common/schema/inter-window-messages';

/**
 * Service that provides diagram model data for visualization components.
 * Communicates with the main application window to retrieve Kubernetes resource data.
 */
@Injectable({
  providedIn: 'root',
})
export class DiagramModelFrameStore {
  private readonly connector = inject(WindowConnectorService);

  /**
   * Stream of diagram models that can be subscribed to for rendering
   * Kubernetes resource visualizations.
   */
  public readonly diagramModel = new ReplaySubject<DiagramModel>(1);

  public readonly currentFrameIndex = new ReplaySubject<number>(1);

  public readonly currentDiagramFrame = combineLatest([
    this.diagramModel,
    this.currentFrameIndex,
  ]).pipe(map(([model, index]) => model.frames[index]));

  private readonly animationSubscriptionStopper = new Subject();

  /**
   * Requests diagram data from the main application.
   *
   * @param resolution - Number of timeline frames to request (higher values provide smoother animations)
   */
  public requestDiagramModel(resolution: number) {
    this.connector
      .callRPC(QUERY_CURRENT_INSPECTION_METADATA, {})
      .pipe(
        map((inspectionMeta) => {
          if (inspectionMeta === null) {
            throw new Error(
              'the main page is not opening an inspection data now',
            );
          }
          return inspectionMeta;
        }),
        map((inspectionMeta) => {
          // generate array of timestamps to gain the diagram frame
          const frames = new Array<number>(resolution);
          for (let i = 0; i < resolution; i++) {
            frames[i] =
              inspectionMeta.startTime +
              ((inspectionMeta.endTime - inspectionMeta.startTime) /
                (resolution - 1)) *
                i;
          }
          return frames;
        }),
        switchMap((frameTimes) =>
          combineLatest(
            frameTimes.map((time) =>
              this.connector.callRPC(QUERY_DIAGRAM_DATA, { ts: time }),
            ),
          ),
        ),
      )
      .subscribe((frames) => {
        this.diagramModel.next({
          frames: frames.map((f, index) => {
            if (f === null) {
              throw new Error(
                `the main page didn't responded a diagram frame at index ${index}`,
              );
            }
            return f.model;
          }),
        });
        this.currentFrameIndex.next(0);
      });
  }

  public setFrameIndex(index: number) {
    this.diagramModel.pipe(take(1)).subscribe((model) => {
      if (model.frames.length > index && index >= 0) {
        this.currentFrameIndex.next(index);
      }
    });
  }

  public incrementFrameIndex() {
    this.currentFrameIndex
      .pipe(withLatestFrom(this.diagramModel), take(1))
      .subscribe(([index, model]) => {
        if (model.frames.length > index + 1) {
          this.currentFrameIndex.next(index + 1);
        }
      });
  }

  public decrementFrameIndex() {
    this.currentFrameIndex.pipe(take(1)).subscribe((index) => {
      if (index > 0) {
        this.currentFrameIndex.next(index - 1);
      }
    });
  }

  public setAnimator(animator: Observable<number>) {
    animator
      .pipe(takeUntil(this.animationSubscriptionStopper))
      .subscribe((index) => {
        this.setFrameIndex(index);
      });
  }

  public stopAnimation() {
    this.animationSubscriptionStopper.next(undefined);
  }
}
