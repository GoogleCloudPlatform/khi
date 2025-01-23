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

/**
 * Returns the default value when the passed value was empty or null.
 * @param value
 * @param defaultValue
 * @returns
 */
export function nonEmptyOrDefaultString(
  value: string,
  defaultValue: string,
): string {
  if (value === undefined || value == null || value.trim() == '') {
    return defaultValue;
  }
  return value;
}
