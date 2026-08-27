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
 * Options for resolving a parameter value from query parameter, candidate overrides, and a default value.
 */
export interface ResolveParamOptions<T> {
  /** Optional URL query parameter name to inspect (e.g. 'downloadChunkSize'). */
  readonly paramName?: string;
  /** Function to parse raw query parameter string into T, returning undefined if invalid. */
  readonly parser?: (rawValue: string) => T | undefined;
  /** Optional URL search query string (defaults to window.location.search when available). */
  readonly searchString?: string;
  /** Candidate values in priority order (e.g. [callerOption, envValue]). */
  readonly candidates?: readonly (T | undefined | null)[];
  /** Mandatory fallback default value. */
  readonly defaultValue: T;
  /** Optional validator function to ensure candidate value is valid before accepting. */
  readonly validator?: (value: T) => boolean;
}

/**
 * Resolves a value by checking a cascade of candidate values in order.
 *
 * Returns the first candidate that is not undefined and not null (and passes validator if provided).
 * Returns the defaultValue if no candidate is valid.
 */
export function resolveConfigValue<T>(
  candidates: readonly (T | undefined | null)[],
  defaultValue: T,
  validator?: (value: T) => boolean,
): T {
  for (const candidate of candidates) {
    if (candidate !== undefined && candidate !== null) {
      if (!validator || validator(candidate)) {
        return candidate;
      }
    }
  }
  return defaultValue;
}

/**
 * Retrieves and parses a query parameter from a URL search string.
 *
 * Returns undefined if searchString is empty, the parameter is not present, or the parser returns undefined.
 */
export function getQueryParam<T>(
  paramName: string,
  parser: (rawValue: string) => T | undefined,
  searchString?: string,
): T | undefined {
  let search = searchString;
  if (
    search === undefined &&
    typeof window !== 'undefined' &&
    window.location?.search
  ) {
    search = window.location.search;
  }
  if (!search) {
    return undefined;
  }
  const params = new URLSearchParams(search);
  const raw = params.get(paramName);
  if (raw === null) {
    return undefined;
  }
  return parser(raw);
}

/**
 * Resolves a parameter value from query parameters, candidate overrides, and a default value.
 *
 * Precedence:
 * 1. Query parameter value (if paramName and parser are provided and query parameter exists).
 * 2. Candidate values in provided order.
 * 3. Default value.
 */
export function resolveParam<T>(options: ResolveParamOptions<T>): T {
  const queryVal =
    options.paramName && options.parser
      ? getQueryParam(options.paramName, options.parser, options.searchString)
      : undefined;
  return resolveConfigValue(
    [queryVal, ...(options.candidates ?? [])],
    options.defaultValue,
    options.validator,
  );
}

/**
 * Parses a boolean string value from query parameters.
 *
 * Accepts 'true', '1', 'yes' as true, and 'false', '0', 'no' as false (case-insensitive).
 * Returns undefined if input is not a recognized boolean representation.
 */
export function parseBooleanQueryParam(
  input: string | null | undefined,
): boolean | undefined {
  if (input === null || input === undefined) {
    return undefined;
  }
  const normalized = input.trim().toLowerCase();
  if (normalized === 'true' || normalized === '1' || normalized === 'yes') {
    return true;
  }
  if (normalized === 'false' || normalized === '0' || normalized === 'no') {
    return false;
  }
  return undefined;
}

/**
 * Parses an integer string value from query parameters.
 *
 * Returns parsed integer if valid and >= min (default 1).
 * Returns undefined otherwise.
 */
export function parseIntegerQueryParam(
  input: string | null | undefined,
  min = 1,
): number | undefined {
  if (input === null || input === undefined) {
    return undefined;
  }
  const parsed = parseInt(input.trim(), 10);
  if (isNaN(parsed) || parsed < min) {
    return undefined;
  }
  return parsed;
}

/**
 * Multipliers for byte units (binary 1024-based).
 */
const BYTE_UNIT_MULTIPLIERS: Readonly<Record<string, number>> = {
  b: 1,
  k: 1024,
  kb: 1024,
  kib: 1024,
  m: 1024 * 1024,
  mb: 1024 * 1024,
  mib: 1024 * 1024,
  g: 1024 * 1024 * 1024,
  gb: 1024 * 1024 * 1024,
  gib: 1024 * 1024 * 1024,
};

/**
 * Parses a byte size string with optional unit suffix into bytes.
 *
 * Examples of valid inputs:
 * - '1048576' -> 1048576
 * - '512KB' / '512KiB' -> 524288
 * - '2MB' / '2MiB' -> 2097152
 * - '1GB' / '1GiB' -> 1073741824
 *
 * Returns undefined if invalid or below min (default 1).
 */
export function parseByteSizeQueryParam(
  input: string | null | undefined,
  min = 1,
): number | undefined {
  if (input === null || input === undefined) {
    return undefined;
  }
  const trimmed = input.trim();
  if (trimmed === '') {
    return undefined;
  }

  // Regex pattern matching numeric prefix with optional unit suffix
  const match = /^([0-9]+(?:\.[0-9]+)?)\s*([a-zA-Z]*)$/.exec(trimmed);
  if (!match) {
    return undefined;
  }

  const num = parseFloat(match[1]);
  if (isNaN(num) || num <= 0) {
    return undefined;
  }

  const unit = match[2].toLowerCase();
  let multiplier = 1;
  if (unit !== '') {
    const found = BYTE_UNIT_MULTIPLIERS[unit];
    if (found === undefined) {
      return undefined;
    }
    multiplier = found;
  }

  const result = Math.floor(num * multiplier);
  if (result < min) {
    return undefined;
  }
  return result;
}
