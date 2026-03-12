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

import { calculateDateLabels } from './date-label-calculator';

describe('calculateDateLabels', () => {
  const DAY = 24 * 60 * 60 * 1000;

  it('should calculate correct basic UTC day boundaries', () => {
    // 2026-03-12T00:00:00Z is 1773273600000
    const leftEdgeTime = new Date('2026-03-11T12:00:00Z').getTime();
    const visibleDuration = DAY * 2; // 2 days visible
    const pixelsPerMs = 1 / 1000 / 60; // 1 pixel per minute

    const labels = calculateDateLabels(
      leftEdgeTime,
      visibleDuration,
      pixelsPerMs,
      0,
    );

    expect(labels.length).toBe(3);

    // Boundary 1: 2026-03-11T00:00:00Z (before left edge)
    expect(labels[0].labelLeft).toBe('2026/03/10');
    expect(labels[0].labelRight).toBe('2026/03/11');
    expect(labels[0].offsetX).toBe(-720);

    // Boundary 2: 2026-03-12T00:00:00Z
    expect(labels[1].labelLeft).toBe('2026/03/11');
    expect(labels[1].labelRight).toBe('2026/03/12');
    expect(labels[1].offsetX).toBe(720);

    // Boundary 3: 2026-03-13T00:00:00Z
    expect(labels[2].labelLeft).toBe('2026/03/12');
    expect(labels[2].labelRight).toBe('2026/03/13');
  });

  it('should calculate correct day boundaries with +9 hours timezone shift', () => {
    // 2026-03-12T00:00:00+09:00 is 2026-03-11T15:00:00Z
    const leftEdgeTime = new Date('2026-03-11T12:00:00Z').getTime(); // 21:00:00+09:00
    const visibleDuration = DAY * 2;
    const pixelsPerMs = 1;

    // Boundary 1: 2026-03-11T00:00:00+09:00 (2026-03-10T15:00:00Z)
    // Boundary 2: 2026-03-12T00:00:00+09:00 (2026-03-11T15:00:00Z)
    const expectedOffsetTime =
      new Date('2026-03-11T15:00:00Z').getTime() - leftEdgeTime;

    const labels = calculateDateLabels(
      leftEdgeTime,
      visibleDuration,
      pixelsPerMs,
      9,
    );

    expect(labels[1].labelLeft).toBe('2026/03/11');
    expect(labels[1].labelRight).toBe('2026/03/12');
    expect(labels[1].offsetX).toBe(expectedOffsetTime * pixelsPerMs);
  });

  it('should calculate correct day boundaries with negative timezone shift (-8 hours)', () => {
    // 2026-03-12T00:00:00-08:00 is 2026-03-12T08:00:00Z
    const leftEdgeTime = new Date('2026-03-12T04:00:00Z').getTime(); // 20:00:00-08:00
    const visibleDuration = DAY * 2;
    const pixelsPerMs = 1;

    // Boundary 1: 2026-03-11T00:00:00-08:00 (2026-03-11T08:00:00Z)
    // Boundary 2: 2026-03-12T00:00:00-08:00 (2026-03-12T08:00:00Z)
    const expectedOffsetTime =
      new Date('2026-03-12T08:00:00Z').getTime() - leftEdgeTime;

    const labels = calculateDateLabels(
      leftEdgeTime,
      visibleDuration,
      pixelsPerMs,
      -8,
    );

    expect(labels[1].labelLeft).toBe('2026/03/11');
    expect(labels[1].labelRight).toBe('2026/03/12');
    expect(labels[1].offsetX).toBe(expectedOffsetTime * pixelsPerMs);
  });
});
