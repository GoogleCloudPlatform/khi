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

import { calculateNewScrollAmount } from './diagram-viewport.component';

describe('calculateNewScrollAmount retain the position in content host space', () => {
  function assertItRetainPositionInContentSpace(
    client: number,
    oldScroll: number,
    viewport: number,
    oldScale: number,
    newScale: number,
  ) {
    const newScrollAmount = calculateNewScrollAmount(
      client,
      oldScroll,
      viewport,
      oldScale,
      newScale,
    );
    expect((client - viewport + oldScroll) / oldScale).toBeCloseTo(
      (client - viewport + newScrollAmount) / newScale,
    );
  }

  it('with basic scale up input', () => {
    assertItRetainPositionInContentSpace(100, 200, 300, 1, 1.5);
  });

  it('with basic scale in input', () => {
    assertItRetainPositionInContentSpace(100, 200, 300, 1, 0.5);
  });
});
