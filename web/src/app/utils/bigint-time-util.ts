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
 * Utility class for converting time represented as BigInt.
 */
export class BigIntTimeUtil {
  /**
   * Converts nanoseconds in BigInt to milliseconds in number, maintaining sub-millisecond precision.
   * @param ns The nanoseconds in BigInt.
   * @returns The milliseconds in number.
   */
  public static NsToNumberMs(ns: bigint): number {
    return Number(ns / 1000000n) + Number(ns % 1000000n) / 1000000;
  }
}
