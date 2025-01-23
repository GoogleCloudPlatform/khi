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

import { TimelineEntry } from '../store/timeline';

/**
 * Filter non used upper layers when there were no any layer lower or equal to given depth
 * @param timelines
 * @param depth
 * @returns
 */
export function SelectOnlyDeeperOrEqual(
  timelines: TimelineEntry[],
  depth: number,
): TimelineEntry[] {
  const result: TimelineEntry[] = [];
  const retainIndicies: Set<number> = new Set();
  for (let i = 0; i < timelines.length; i++) {
    const timeline = timelines[i];
    if (timeline.layer >= depth) {
      let prevLayer = timeline.layer;
      for (let j = 0; j <= i && prevLayer >= 0; j++) {
        if (timelines[i - j].layer == prevLayer) {
          if (retainIndicies.has(i - j)) break;
          retainIndicies.add(i - j);
          prevLayer -= 1;
        }
      }
    }
  }
  for (let i = 0; i < timelines.length; i++) {
    if (retainIndicies.has(i)) {
      result.push(timelines[i]);
    }
  }
  return result;
}
