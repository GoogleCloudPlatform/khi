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

/**
 * Represents the difference status of a segment or line.
 */
export enum DiffStatus {
  Added = 'added',
  Deleted = 'deleted',
  Unchanged = 'unchanged',
  Modified = 'modified',
  MovedOut = 'moved-out',
  MovedIn = 'moved-in',
}

/**
 * Represents a segment of a value, optionally highlighted for character-level differences.
 */
export interface ValueSegment {
  /** The text content of the segment. */
  text: string;
  /** The difference status of this segment. */
  diffStatus: DiffStatus;
}

/**
 * Computes character-level difference between two strings using LCS.
 */
export function diffStrings(
  oldStr: string,
  newStr: string,
): { oldSegs: ValueSegment[]; newSegs: ValueSegment[] } {
  const m = oldStr.length;
  const n = newStr.length;

  const dp: number[][] = Array.from({ length: m + 1 }, () =>
    Array(n + 1).fill(0),
  );

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (oldStr[i - 1] === newStr[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
      }
    }
  }

  const oldSegs: ValueSegment[] = [];
  const newSegs: ValueSegment[] = [];

  let i = m;
  let j = n;

  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && oldStr[i - 1] === newStr[j - 1]) {
      const char = oldStr[i - 1];
      oldSegs.unshift({ text: char, diffStatus: DiffStatus.Unchanged });
      newSegs.unshift({ text: char, diffStatus: DiffStatus.Unchanged });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      newSegs.unshift({ text: newStr[j - 1], diffStatus: DiffStatus.Added });
      j--;
    } else if (i > 0 && (j === 0 || dp[i][j - 1] < dp[i - 1][j])) {
      oldSegs.unshift({ text: oldStr[i - 1], diffStatus: DiffStatus.Deleted });
      i--;
    }
  }

  return {
    oldSegs: mergeAdjacentSegments(oldSegs),
    newSegs: mergeAdjacentSegments(newSegs),
  };
}

/**
 * Merges consecutive segments with the same diff status.
 */
function mergeAdjacentSegments(segs: ValueSegment[]): ValueSegment[] {
  if (segs.length === 0) {
    return [];
  }
  const merged: ValueSegment[] = [segs[0]];
  for (let idx = 1; idx < segs.length; idx++) {
    const last = merged[merged.length - 1];
    const curr = segs[idx];
    if (last.diffStatus === curr.diffStatus) {
      last.text += curr.text;
    } else {
      merged.push(curr);
    }
  }
  return merged;
}
