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
 * Checks if the delta represents an addition.
 * An addition delta is an array containing only the new value: [newValue].
 */
export function isAddedDelta(delta: unknown): delta is [unknown] {
  return Array.isArray(delta) && delta.length === 1;
}

/**
 * Checks if the delta represents a modification.
 * A modification delta is an array containing the old value and the new value: [oldValue, newValue].
 */
export function isModifiedDelta(delta: unknown): delta is [unknown, unknown] {
  return Array.isArray(delta) && delta.length === 2;
}

/**
 * Checks if the delta represents a deletion.
 * A deletion delta is an array containing the old value, 0, and 0: [oldValue, 0, 0].
 */
export function isDeletedDelta(
  delta: unknown,
): delta is [unknown, unknown, number] {
  return Array.isArray(delta) && delta.length === 3 && delta[2] === 0;
}

/**
 * Checks if the delta represents an array element move.
 * A move delta is an array containing the old value, the destination index, and 3: [oldValue, destIndex, 3].
 */
export function isMovedDelta(
  delta: unknown,
): delta is [unknown, number, number] {
  return Array.isArray(delta) && delta.length === 3 && delta[2] === 3;
}
