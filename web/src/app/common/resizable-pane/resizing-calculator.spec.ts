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

import { take, toArray } from 'rxjs';
import { ResizableArea, ResizingCalculator } from './resizing-calculator';

function verifyResizingByContainerSize(
  areas: ResizableArea[],
  containerSize: number,
) {
  const calculator = new ResizingCalculator(areas);
  calculator.setContainerSizeInPx(containerSize);
  return calculator.areaStates;
}

function verifyResizingByAreaSizeChange(
  areas: ResizableArea[],
  containerSize: number,
  resizeTarget: string,
  resizeTargetSize: number,
  ignoreMinimumSize: boolean,
) {
  const calculator = new ResizingCalculator(areas);
  calculator.setContainerSizeInPx(containerSize);
  calculator.resizeArea(resizeTarget, resizeTargetSize, ignoreMinimumSize);
  return calculator.areaStates;
}

describe('ResizingCalculator', () => {
  it('returns an error when user gave non integer resizeRatio to ResizingCalculator', () => {
    expect(() => {
      new ResizingCalculator([
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 2,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1.5,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1.5,
        },
      ]);
    }).toThrowError();
  });
  it('does not resize areas when sum of current given area sizes matches the container size', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 100, 100]);
      done();
    });
  });

  it('expands areas when container size is bigger than current size with same expand ratio ', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      400,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 150, 150]);
      expect(states.reduce((a, b) => a + b.currentSize, 0)).toEqual(400);
      done();
    });
  });

  it('expands areas when container size is bigger than initial size with different expand ratio ', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 3,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      400,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 175, 125]);
      expect(states.reduce((a, b) => a + b.currentSize, 0)).toBe(400);
      done();
    });
  });

  it('expands areas when the container is bigger than current sum of area sizes and the resize amount is not divisable', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 2,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      400,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 166, 134]);
      expect(states.reduce((a, b) => a + b.currentSize, 0)).toBe(400);
      done();
    });
  });

  it('shrinks areas when container size is smaller than current sum of area sizes with same resizeRatio', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      200,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 50, 50]);
      done();
    });
  });

  it('shrinks areas when container size is smaller than current sum of area sizes with different resizeRatio ', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 3,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      200,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 25, 75]);
      done();
    });
  });

  it('shrinks areas when container size is smaller than current sum of area sizes and the shrinking operation could reach minSize of an area before acheiving the size ', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 80,
          resizeRatio: 1,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      200,
    ).subscribe((states) => {
      // Resize ratio is same, but the result size is different because of minSizeInPx
      expect(states.map((state) => state.currentSize)).toEqual([100, 80, 20]);
      done();
    });
  });

  it('shrinks areas to the minimum when the given container size is not feasible with keeping the minSizePx of areas', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 100,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 50,
          resizeRatio: 1,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 50,
          resizeRatio: 1,
        },
      ],
      150,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 50, 50]);
      done();
    });
  });

  it('shrinks areas to the minimum when the given container size is not feasible with keeping the minSizePx of areas with the different minimum size', (done) => {
    verifyResizingByContainerSize(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 100,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 20,
          resizeRatio: 1,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 50,
          resizeRatio: 1,
        },
      ],
      150,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 20, 50]);
      done();
    });
  });

  it('shrinks specified area and expand other if it can', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
      'c',
      90,
      false,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([110, 100, 90]);
      done();
    });
  });

  it('does not shrinks specified area when others can not be expanded', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
      'c',
      90,
      false,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 100, 100]);
      done();
    });
  });

  it('shrinks with ignoreing the minimum size when ignoreMinimumSize flag is true', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 50,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
      ],
      300,
      'b',
      0,
      true,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([150, 0, 150]);
      done();
    });
  });

  it('expands specified area with shrinking others', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 1,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
      'c',
      110,
      false,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([90, 100, 110]);
      done();
    });
  });

  it('does not expand specified area when others can not be expanded', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
      'c',
      110,
      false,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([100, 100, 100]);
      done();
    });
  });

  it('expands the specified area with shrinking others with the minimum size limit', (done) => {
    verifyResizingByAreaSizeChange(
      [
        {
          id: 'a',
          initialSize: 100,
          minSizeInPx: 95,
          resizeRatio: 1,
        },
        {
          id: 'b',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
        {
          id: 'c',
          initialSize: 100,
          minSizeInPx: 0,
          resizeRatio: 0,
        },
      ],
      300,
      'c',
      110,
      false,
    ).subscribe((states) => {
      expect(states.map((state) => state.currentSize)).toEqual([95, 100, 105]);
      done();
    });
  });

  it('emits the new ResizableArea when container size is changed', (done) => {
    const r = new ResizingCalculator([
      {
        id: 'a',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 0,
      },
      {
        id: 'b',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
      {
        id: 'c',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
    ]);
    r.areaStates.pipe(take(4), toArray()).subscribe((states) => {
      expect(states[0].map((state) => state.currentSize)).toEqual([
        100, 100, 100,
      ]);
      expect(states[1].map((state) => state.currentSize)).toEqual([
        100, 150, 150,
      ]);
      expect(states[2].map((state) => state.currentSize)).toEqual([
        100, 200, 200,
      ]);
      expect(states[3].map((state) => state.currentSize)).toEqual([
        100, 25, 25,
      ]);
      done();
    });
    r.setContainerSizeInPx(400);
    r.setContainerSizeInPx(500);
    r.setContainerSizeInPx(150);
  });

  it('emits sizes of specified areas when container size is changed', (done) => {
    const r = new ResizingCalculator([
      {
        id: 'a',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 0,
      },
      {
        id: 'b',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
      {
        id: 'c',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
    ]);
    r.areaSize('b')
      .pipe(take(4), toArray())
      .subscribe((states) => {
        expect(states).toEqual([100, 150, 200, 25]);
        done();
      });
    r.setContainerSizeInPx(400);
    r.setContainerSizeInPx(500);
    r.setContainerSizeInPx(150);
  });

  it('emits sizes of specified areas when container size is changed and only when its size changed', (done) => {
    const r = new ResizingCalculator([
      {
        id: 'a',
        initialSize: 100,
        minSizeInPx: 80,
        resizeRatio: 1,
      },
      {
        id: 'b',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
      {
        id: 'c',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
    ]);
    r.areaSize('a')
      .pipe(take(4), toArray())
      .subscribe((states) => {
        expect(states).toEqual([100, 90, 80, 90]);
        done();
      });
    r.setContainerSizeInPx(270);
    r.setContainerSizeInPx(100);
    r.setContainerSizeInPx(90);
    r.setContainerSizeInPx(120);
  });

  it('returns current specified area size with getAreaSize', () => {
    const r = new ResizingCalculator([
      {
        id: 'a',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
      {
        id: 'b',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
      {
        id: 'c',
        initialSize: 100,
        minSizeInPx: 0,
        resizeRatio: 1,
      },
    ]);
    r.setContainerSizeInPx(300);
    expect(r.getAreaSize('a')).toBe(100);
    expect(r.getAreaSize('b')).toBe(100);
    expect(r.getAreaSize('c')).toBe(100);
    r.setContainerSizeInPx(360);
    expect(r.getAreaSize('a')).toBe(120);
    expect(r.getAreaSize('b')).toBe(120);
    expect(r.getAreaSize('c')).toBe(120);
  });
});
