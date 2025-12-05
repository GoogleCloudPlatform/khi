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

/**
 * Asserts that the given value is never.
 *
 * This function is used for making sure the code is unreachable with Typescript type check.
 * Example:
 * let a: "A" | "B";
 * switch (a) {
 *   case "A":
 *     break;
 *   case "B":
 *     break;
 *   default:
 *     unreachable(a); // this won't be an error.
 * }
 *
 * switch(a) {
 *   case "A":
 *     break;
 *   default:
 *     unreachable(a); // this will be an error because the 'a' is "B" and not never.
 * }
 * @param v The value to assert.
 */
export function unreachable(v: never): never {
  console.error('unreachable code reached', v);
  throw new Error(`unreachable code reached with value: ${JSON.stringify(v)}`);
}
