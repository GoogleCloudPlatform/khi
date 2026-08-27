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

import { environment } from 'src/environments/environment';
import {
  parseBooleanQueryParam,
  resolveParam,
} from 'src/app/utils/config-resolver';

/**
 * Options supplied to resolve the effective transport binary format setting.
 */
export interface ResolveTransportConfigOptions {
  /** Optional URL search query string (defaults to window.location.search when available). */
  readonly searchString?: string;
  /** Optional explicit boolean override or environment value. */
  readonly environmentUseBinaryFormat?: boolean;
}

/**
 * Resolves whether Connect-RPC transport should use binary format (protobuf).
 *
 * Priority order:
 * 1. URL GET parameter `useBinaryFormat` (e.g. `?useBinaryFormat=true` or `?useBinaryFormat=0`).
 * 2. Environment configuration (`environment.useBinaryFormat`).
 * 3. Default fallback (`false`).
 */
export function resolveUseBinaryFormat(
  options?: ResolveTransportConfigOptions,
): boolean {
  return resolveParam({
    paramName: 'useBinaryFormat',
    parser: parseBooleanQueryParam,
    searchString: options?.searchString,
    candidates: [
      options?.environmentUseBinaryFormat,
      environment.useBinaryFormat,
    ],
    defaultValue: false,
  });
}
