/**
 * Copyright 2026 Google LLC
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

import { GraphRoot } from './graph-root';
import { graphRootIt } from './test/graph-test-utiility';

function setGraphRootContainerSize(
  gr: GraphRoot,
  width: number,
  height: number,
): void {
  (
    gr as unknown as { _currentSize: { width: number; height: number } }
  )._currentSize = {
    width,
    height,
  };
}

describe('GraphRoot', () => {
  graphRootIt('should fit and center bounds in viewBox', (gr) => {
    setGraphRootContainerSize(gr, 1000, 800);

    gr.fitBounds(100, 100, 500, 400, 20);

    const viewBox = gr.element?.getAttribute('viewBox');
    expect(viewBox).toBeTruthy();
    const [vx, vy, vw, vh] = viewBox!.split(',').map(Number);
    expect(vw).toBe(1000);
    expect(vh).toBe(800);
    // Center of 100..600 with padding 20 is at 350. View width 1000 centered is 350 - 500 = -150.
    expect(vx).toBe(-150);
    // Center of 100..500 with padding 20 is at 300. View height 800 centered is 300 - 400 = -100.
    expect(vy).toBe(-100);
  });

  graphRootIt(
    'should scale down when graph bounds exceed container size',
    (gr) => {
      setGraphRootContainerSize(gr, 500, 400);

      gr.fitBounds(0, 0, 1000, 600, 20);

      const viewBox = gr.element?.getAttribute('viewBox');
      expect(viewBox).toBeTruthy();
      const [vx, vy, vw, vh] = viewBox!.split(',').map(Number);
      expect(vw).toBe(1040);
      expect(vh).toBe(832);
      expect(vx).toBe(-20);
      expect(vy).toBe(-116);
    },
  );

  graphRootIt('should return early when dimensions are non-positive', (gr) => {
    setGraphRootContainerSize(gr, 1000, 800);

    gr.fitBounds(0, 0, 0, 0);
    expect(gr.element?.getAttribute('viewBox')).toBeNull();
  });
});
