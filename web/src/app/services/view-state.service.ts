/**
 * Copyright 2024 Google LLC
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

import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  Subject,
  animationFrames,
  distinctUntilChanged,
  map,
  shareReplay,
} from 'rxjs';

/**
 * A service to manage statuses used for view in application wide.
 */
@Injectable({ providedIn: 'root' })
export class ViewStateService {
  /**
   * Rendering small sharp shapes with WebGL in the default pixel ratio can be blurry result in high resolution display like Mac's retina display.
   * Higher value seems to cause scaling issue on high resoluition display because it can easily hit the maximum canvas width.
   * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/canvas#maximum_canvas_size
   */
  public static DEVICE_PIXEL_RATIO_SCALE = 1;

  private timezoneShiftSubject: BehaviorSubject<number> = new BehaviorSubject(
    -new Date().getTimezoneOffset() / 60,
  );

  /**
   * Number of the hours differences from UTC
   */
  public timezoneShift: Observable<number> = this.timezoneShiftSubject;

  private timelineStateResetCommandSubject: Subject<null> = new Subject();

  /**
   * Emit value when timeline scale and offset reset was requesed.
   */
  public timelineStateResetCommand: Observable<null> =
    this.timelineStateResetCommandSubject;

  private timeOffsetSubject = new BehaviorSubject(0);

  public timeOffset: Observable<number> = this.timeOffsetSubject;

  private pixelPerTimeSubject = new BehaviorSubject<number>(0.01);

  public pixelPerTime: Observable<number> = this.pixelPerTimeSubject;

  private visibleWidthSubject = new BehaviorSubject(100);

  public visibleWidth: Observable<number> = this.visibleWidthSubject.pipe(
    shareReplay(1),
  );

  /**
   * Whether KHI hides a subresource layer timeline without any matching with the log filter.
   */
  public hideSubresourcesWithoutMatchingLogs = new BehaviorSubject(false);

  /**
   * Whether KHI hides a resource layer timeline without any matching with the log filter.
   */
  public hideResourcesWithoutMatchingLogs = new BehaviorSubject(true);

  public devicePixelRatio = animationFrames().pipe(
    map(
      () => window.devicePixelRatio * ViewStateService.DEVICE_PIXEL_RATIO_SCALE,
    ),
    distinctUntilChanged(),
    shareReplay(1),
  );

  public setTimezoneShift(timezoneShift: number): void {
    this.timezoneShiftSubject.next(timezoneShift);
  }

  /**
   * Set the offset of timeline view
   * @param offset offset time of left most edge of the timeline view
   */
  public setTimeOffset(offset: number): void {
    this.timeOffsetSubject.next(offset);
  }

  public getTimeOffset(): number {
    return this.timeOffsetSubject.value;
  }

  public setPixelPerTime(pixelPerTime: number): void {
    return this.pixelPerTimeSubject.next(pixelPerTime);
  }

  public getPixelPerTime(): number {
    return this.pixelPerTimeSubject.value;
  }

  public setVisibleWidth(widthInPx: number): void {
    this.visibleWidthSubject.next(widthInPx);
  }

  public getVisibleWidth(): number {
    return this.visibleWidthSubject.value;
  }

  public setHideSubresourcesWithoutMatchingLogs(
    hideSubresourcesWithoutMatchingLogs: boolean,
  ): void {
    this.hideSubresourcesWithoutMatchingLogs.next(
      hideSubresourcesWithoutMatchingLogs,
    );
  }

  public setHideResourcesWithoutMatchingLogs(
    hideREsourcesWithoutMatchingLogs: boolean,
  ): void {
    this.hideResourcesWithoutMatchingLogs.next(
      hideREsourcesWithoutMatchingLogs,
    );
  }

  /**
   * Reset scale and offset of timeline status.
   */
  public resetTimelineStatus(): void {
    this.timelineStateResetCommandSubject.next(null);
  }
}
