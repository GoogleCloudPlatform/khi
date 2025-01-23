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

export function iterToArr<T>(iter: IterableIterator<T>): T[] {
  const result: T[] = [];
  for (const elem of iter) {
    result.push(elem);
  }
  return result;
}

export function subtractSet<T>(
  baseSet: Set<T>,
  subtractingSet: Set<T>,
): Set<T> {
  const result = new Set<T>();
  for (const elem of baseSet.values()) {
    if (!subtractingSet.has(elem)) {
      result.add(elem);
    }
  }
  return result;
}

export function filteElementsByIncludedSubstring(
  candidates: Iterable<string>,
  query: string,
): string[] {
  const middleResult: { value: string; index: number }[] = [];
  for (const candidate of candidates) {
    const index = candidate.indexOf(query);
    if (index != -1) {
      middleResult.push({
        value: candidate,
        index,
      });
    }
  }
  return middleResult
    .sort((a, b) => {
      const diff = a.index - b.index;
      return diff == 0 ? a.value.localeCompare(b.value) : diff;
    })
    .map((a) => a.value);
}
