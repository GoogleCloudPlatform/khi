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

import {
  BehaviorSubject,
  distinctUntilChanged,
  map,
  Observable,
  of,
  switchMap,
  throwError,
} from 'rxjs';

/**
 * ResizableArea is the definition of the area calculated by ResizingCalculator.
 */
export interface ResizableArea {
  id: string;
  initialSize: number;
  minSizeInPx: number;
  // the ratio to expand/shrink. This value must be integer.
  resizeRatio: number;
}

interface ResizableAreaState {
  spec: ResizableArea;
  currentSize: number;
}

/**
 * informations needed to be calculated before shrinking operation in ResizingCalculator.
 */
interface ShrinkableAreaInfo {
  /**
   * Id of given ResizableArea array where is the first area reaching the minSizeInPx.
   */
  nextShrinkAreaId: string;
  /**
   * The maximum size per ratio not violating the limit in nextShrinkArea.
   */
  shrinkableSizePerRatioToReachTheLimit: number;
  /**
   * The sum of resize ratio in given array. This will ignore area already reaching the limits.
   */
  sumOfShrinkableResizeRatio: number;
  /**
   * The maximum shrinkable size before reaching the limit.
   */
  maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea: number;
}

export class ResizingCalculator {
  public areaStates: BehaviorSubject<ResizableAreaState[]>;

  constructor(areas: ResizableArea[]) {
    if (areas.find((area) => !Number.isInteger(area.resizeRatio))) {
      throw new Error(
        'resizeRatio must be interger but floating value was given',
      );
    }
    this.areaStates = new BehaviorSubject(
      areas.map((a) => ({ spec: a, currentSize: a.initialSize })),
    );
  }

  public setContainerSizeInPx(size: number) {
    this.areaStates.next(
      this.constraintByContainerSize(this.areaStates.value, size),
    );
  }

  public areaSize(id: string): Observable<number> {
    return this.areaStates.pipe(
      // monitor only the specified area
      switchMap((areas) => {
        const area = areas.find((area) => area.spec.id === id);
        if (area) {
          return of(area);
        } else {
          return throwError(() => `area id ${id} not found`);
        }
      }),
      map((area) => area.currentSize),
      distinctUntilChanged(),
    );
  }

  /**
   * Return the size of specific area.
   */
  public getAreaSize(id: string): number {
    return (
      this.areaStates.value.find((area) => area.spec.id === id)?.currentSize ||
      0
    );
  }

  /**
   * Resize specified area size to the given size.
   */
  public resizeArea(
    id: string,
    sizeInPx: number,
    ignoreMinimumSize: boolean = false,
  ): void {
    const areas = this.areaStates.value.map((area) => ({
      spec: area.spec,
      currentSize: area.currentSize,
    }));
    const resizeTarget = areas.find((area) => area.spec.id === id);
    if (!resizeTarget) {
      return;
    }
    if (resizeTarget.currentSize === sizeInPx) {
      return;
    }
    let restSizeDiff = sizeInPx - resizeTarget.currentSize;
    if (restSizeDiff > 0) {
      resizeTarget.currentSize = sizeInPx;
      // expand specified area, shrink others
      for (let i = 0; i < 100; i++) {
        // calculate shrink info without the expanding target
        const shrinkInfo = this.calculateShrinkableAreaInfo(
          areas.filter((area) => area.spec.id !== id),
        );
        if (shrinkInfo.nextShrinkAreaId === '') {
          resizeTarget.currentSize -= restSizeDiff;
          break;
        }
        if (
          restSizeDiff <
          shrinkInfo.maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea
        ) {
          // Resize can be done without reaching the minSize
          let restSumOfShrinkRatio = shrinkInfo.sumOfShrinkableResizeRatio;
          for (const area of areas) {
            if (area.spec.resizeRatio === 0) continue;
            if (area.spec.id === id) continue;
            if (area.spec.minSizeInPx === area.currentSize) continue;
            const shrinkSize = Math.floor(
              (restSizeDiff / restSumOfShrinkRatio) * area.spec.resizeRatio,
            );
            area.currentSize -= shrinkSize;
            restSizeDiff -= shrinkSize;
            restSumOfShrinkRatio -= area.spec.resizeRatio;
          }
          break;
        } else {
          const areaReachingTheLimit = areas.find(
            (area) => area.spec.id === shrinkInfo.nextShrinkAreaId,
          )!;
          const diff =
            areaReachingTheLimit.currentSize -
            areaReachingTheLimit.spec.minSizeInPx;
          restSizeDiff -= diff;
          areaReachingTheLimit.currentSize =
            areaReachingTheLimit.spec.minSizeInPx;
          let restSumOfShrinkRatio =
            shrinkInfo.sumOfShrinkableResizeRatio -
            areaReachingTheLimit.spec.resizeRatio;
          let restSizeDiffInThisLoop =
            shrinkInfo.maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea -
            diff;
          for (const area of areas) {
            if (area.spec.resizeRatio === 0) continue;
            if (area.spec.id === id) continue;
            if (area.spec.minSizeInPx === area.currentSize) continue;
            const shrinkSize = Math.floor(
              (restSizeDiffInThisLoop / restSumOfShrinkRatio) *
                area.spec.resizeRatio,
            );
            area.currentSize -= shrinkSize;
            restSizeDiff -= shrinkSize;
            restSizeDiffInThisLoop -= shrinkSize;
            restSumOfShrinkRatio -= area.spec.resizeRatio;
          }
        }
      }
    } else {
      // shrink specified area, expand others
      if (sizeInPx < resizeTarget.spec.minSizeInPx && !ignoreMinimumSize) {
        sizeInPx = resizeTarget.spec.minSizeInPx;
        restSizeDiff = resizeTarget.currentSize - sizeInPx;
      }
      restSizeDiff = -restSizeDiff;
      let restSumOfExpandRatio = areas.reduce(
        (a, b) => a + b.spec.resizeRatio,
        0,
      );
      restSumOfExpandRatio -= resizeTarget.spec.resizeRatio;
      for (const area of areas) {
        if (area.spec.resizeRatio === 0) continue;
        if (area.spec.id === id) continue;
        const expandSize = Math.floor(
          (restSizeDiff / restSumOfExpandRatio) * area.spec.resizeRatio,
        );
        area.currentSize += expandSize;
        restSizeDiff -= expandSize;
        restSumOfExpandRatio -= area.spec.resizeRatio;
      }
      resizeTarget.currentSize = sizeInPx;
      // If the rest of areas are not expandable
      if (restSizeDiff > 0) {
        resizeTarget.currentSize += restSizeDiff;
      }
    }
    this.areaStates.next(areas);
  }

  /**
   * Check the wrapper size then shrink or expand the rest of panes.
   * @param areaSizeState the original pane size not considering the container size.
   */
  private constraintByContainerSize(
    areaSizeState: ResizableAreaState[],
    containerSize: number,
  ): ResizableAreaState[] {
    areaSizeState = areaSizeState.map((area) => ({
      spec: area.spec,
      currentSize: area.currentSize,
    }));
    const currentSize = this.getEntirePaneSize(areaSizeState);
    if (currentSize === containerSize) return areaSizeState;
    const minSize = this.getMinimumEntireAreaSize(areaSizeState);
    // contents can't be fit in the container even with the minimum size. override container size as the last way.
    if (containerSize < minSize) {
      containerSize = minSize;
      for (const area of areaSizeState) {
        area.currentSize = area.spec.minSizeInPx;
      }
      return areaSizeState;
    }

    if (currentSize < containerSize) {
      // Expanding areas
      let restSizeDiff = containerSize - currentSize;
      let restSumOfExpandRatio = areaSizeState.reduce(
        (a, b) => a + b.spec.resizeRatio,
        0,
      );
      for (const area of areaSizeState) {
        if (area.spec.resizeRatio === 0) continue;
        const expandSize = Math.floor(
          (restSizeDiff / restSumOfExpandRatio) * area.spec.resizeRatio,
        );
        area.currentSize += expandSize;
        restSizeDiff -= expandSize;
        restSumOfExpandRatio -= area.spec.resizeRatio;
      }
      return areaSizeState;
    } else {
      // Shrinking area
      // Shrink operation is not straight because each areas can have different minSizeInPx even they had non-zero resizeRatio
      let restSizeDiff = currentSize - containerSize;
      for (let i = 0; i < 100; i++) {
        // Loops until the specific times to avoid infinity loop if bug occurs
        const shrinkInfo = this.calculateShrinkableAreaInfo(areaSizeState);
        // No shrinkable area anymore, but here should be unreachable.
        if (shrinkInfo.nextShrinkAreaId === '') return areaSizeState;
        if (
          restSizeDiff <
          shrinkInfo.maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea
        ) {
          // Resize can be done without reaching the minSize
          let restSumOfShrinkRatio = shrinkInfo.sumOfShrinkableResizeRatio;
          for (const area of areaSizeState) {
            if (area.spec.resizeRatio === 0) continue;
            if (area.spec.minSizeInPx === area.currentSize) continue;
            const shrinkSize = Math.floor(
              (restSizeDiff / restSumOfShrinkRatio) * area.spec.resizeRatio,
            );
            area.currentSize -= shrinkSize;
            restSizeDiff -= shrinkSize;
            restSumOfShrinkRatio -= area.spec.resizeRatio;
          }
          break;
        } else {
          // Resize can't be done without reaching the minSize of the area
          const nextShrinkAreaReachingLimit = areaSizeState.find(
            (area) => area.spec.id === shrinkInfo.nextShrinkAreaId,
          )!;
          const diff =
            nextShrinkAreaReachingLimit.currentSize -
            nextShrinkAreaReachingLimit.spec.minSizeInPx;
          restSizeDiff -= diff;
          nextShrinkAreaReachingLimit.currentSize =
            nextShrinkAreaReachingLimit.spec.minSizeInPx;
          let restSumOfShrinkRatio =
            shrinkInfo.sumOfShrinkableResizeRatio -
            nextShrinkAreaReachingLimit.spec.resizeRatio;
          let restSizeDiffInThisLoop =
            shrinkInfo.maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea -
            diff;
          for (const area of areaSizeState) {
            if (area.spec.resizeRatio === 0) continue;
            if (area.spec.minSizeInPx === area.currentSize) continue;
            const shrinkSize = Math.floor(
              (restSizeDiffInThisLoop / restSumOfShrinkRatio) *
                area.spec.resizeRatio,
            );
            area.currentSize -= shrinkSize;
            restSizeDiff -= shrinkSize;
            restSizeDiffInThisLoop -= shrinkSize;
            restSumOfShrinkRatio -= area.spec.resizeRatio;
          }
        }
      }
      return areaSizeState;
    }
  }

  private getEntirePaneSize(areas: ResizableAreaState[]): number {
    return areas.reduce((a, b) => a + b.currentSize, 0);
  }

  /**
   * Calculate ShrinkableAreaInfo before performing actual resize.
   */
  private calculateShrinkableAreaInfo(
    areas: ResizableAreaState[],
  ): ShrinkableAreaInfo {
    let minShrinkableSizePerRatio = 100000;
    let resultIndex = -1;
    let sumOfShrinkableAreaResizeRatio = 0;
    for (let areaIndex = 0; areaIndex < areas.length; areaIndex++) {
      if (areas[areaIndex].spec.resizeRatio === 0) continue;
      // Ignore area already reached the minimum size
      if (areas[areaIndex].currentSize === areas[areaIndex].spec.minSizeInPx)
        continue;
      sumOfShrinkableAreaResizeRatio += areas[areaIndex].spec.resizeRatio;
      const shrinkableSizePerRatio =
        (areas[areaIndex].currentSize - areas[areaIndex].spec.minSizeInPx) /
        areas[areaIndex].spec.resizeRatio;
      if (shrinkableSizePerRatio < minShrinkableSizePerRatio) {
        resultIndex = areaIndex;
        minShrinkableSizePerRatio = shrinkableSizePerRatio;
      }
    }
    return {
      nextShrinkAreaId: resultIndex === -1 ? '' : areas[resultIndex].spec.id,
      shrinkableSizePerRatioToReachTheLimit: minShrinkableSizePerRatio,
      sumOfShrinkableResizeRatio: sumOfShrinkableAreaResizeRatio,
      maxShrinkableSizeBeforeReachingMinSizeOfMinShrinkableArea:
        sumOfShrinkableAreaResizeRatio * minShrinkableSizePerRatio,
    };
  }

  /**
   * Returns sum of the minimum area size. Parent container can't be smaller than this.
   * @param areas whole areas included in the parent container
   * @returns sum of the minimum area size
   */
  private getMinimumEntireAreaSize(areas: ResizableAreaState[]): number {
    return areas.reduce((a, b) => a + b.spec.minSizeInPx, 0);
  }
}
