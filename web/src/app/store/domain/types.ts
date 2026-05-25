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
 * Deeply readonly type to ensure domain objects cannot be mutated by the view.
 */
// eslint-disable-next-line @typescript-eslint/no-unsafe-function-type
export type ReadonlyDomainElement<T> = T extends Function
  ? T
  : T extends readonly (infer U)[]
    ? readonly ReadonlyDomainElement<U>[]
    : T extends object
      ? { readonly [K in keyof T]: ReadonlyDomainElement<T[K]> }
      : T;

/**
 * Type representing a value that can be undefined.
 */
export type Undefinable<T> = T | undefined;
