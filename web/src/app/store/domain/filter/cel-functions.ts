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

import { CELLog, CELTimeline } from './cel-types';

const SEVERITY_LEVELS = {
  UNKNOWN: 0n,
  INFO: 1n,
  WARNING: 2n,
  ERROR: 3n,
  FATAL: 4n,
};

/**
 * Maps severity label to its corresponding numerical representation for comparison in CEL.
 *
 * @param label - The severity label string
 * @returns BigInt value representing the severity level
 */
export function mapSeverityToNumber(label: string | undefined): bigint {
  if (!label) return SEVERITY_LEVELS.UNKNOWN;
  const upper = label.toUpperCase();
  if (upper.includes('FATAL')) return SEVERITY_LEVELS.FATAL;
  if (upper.includes('ERROR')) return SEVERITY_LEVELS.ERROR;
  if (upper.includes('WARN')) return SEVERITY_LEVELS.WARNING;
  if (upper.includes('INFO')) return SEVERITY_LEVELS.INFO;
  return SEVERITY_LEVELS.UNKNOWN;
}

/**
 * Handler for matching a timeline field (name, type, or path map field) against a regex pattern or list of patterns case-insensitively.
 *
 * @param t - The timeline instance (Map) passed from the CEL evaluator
 * @param key - The field key (e.g. 'name', 'type', 'kind', 'namespace'). '*' to match any keys.
 * @param val - The expected value or list of values
 * @returns True if any pattern matches the field value case-insensitively
 */
export function matchTimelinePath(
  timeline: CELTimeline | undefined,
  key: string,
  val: string | string[],
): boolean {
  if (!timeline) {
    return false;
  }

  const keyTyped = key as string;
  const valTyped = Array.isArray(val) ? (val as string[]) : [val as string];

  const keyStr = keyTyped.toLowerCase();
  const pathMap = timeline.path as Record<string, string> | undefined;
  if (!pathMap) {
    return false;
  }

  const keys = keyTyped === '*' ? Object.keys(pathMap) : [keyStr];
  return valTyped.some((pat) => {
    try {
      const regex = new RegExp(String(pat), 'i');
      return keys.some((key) => key in pathMap && regex.test(pathMap[key]));
    } catch {
      return false;
    }
  });
}

/**
 * Matches a dot-separated log body key path against a pattern or list of patterns case-insensitively.
 *
 * Traverses the structured log body object using a dot-separated path (e.g., "a.b.c"). Specify '*' to match any field.
 * If the nested path exists and has a value, it matches the value against the expected patterns.
 *
 * @param l - The log instance passed from the CEL evaluator
 * @param pathKey - The dot-separated path key to traverse in the log body
 * @param val - The expected pattern or array of patterns to match against
 * @returns True if any pattern matches the resolved path value case-insensitively.
 */
export function matchLogField(
  l: CELLog | undefined,
  pathKey: string,
  val: string | string[],
): boolean {
  if (!l) {
    return false;
  }

  let actualVal: string;
  if (pathKey === '*') {
    actualVal = l.bodyYAML;
  } else {
    const parts = pathKey.split('.');
    let current: unknown = l.body;

    for (const part of parts) {
      if (
        current === null ||
        current === undefined ||
        typeof current !== 'object' ||
        Array.isArray(current)
      ) {
        return false;
      }
      current = (current as Record<string, unknown>)[part];
    }

    if (current === undefined || current === null) {
      return false;
    }
    if (typeof current === 'object') {
      return false;
    } else {
      actualVal = String(current);
    }
  }
  const patterns = Array.isArray(val) ? val : [val];

  return patterns.some((pat) => {
    try {
      const regex = new RegExp(String(pat), 'i');
      return regex.test(actualVal);
    } catch {
      return false;
    }
  });
}

/**
 * Checks if any revision in the timeline has a body field matching the pattern.
 *
 * @param t - The timeline instance (Map) passed from the CEL evaluator
 * @param pathKey - The dot-separated path key to traverse in the revision body
 * @param val - The expected pattern or array of patterns to match against
 * @returns True if any revision has a matching body field.
 */
export function matchTimelineRevisionBodyField(
  t: CELTimeline | undefined,
  pathKey: string,
  val: string | string[],
): boolean {
  if (!t) {
    return false;
  }

  const parts = pathKey.split('.');
  return t.revisions.some((r) => {
    let actualVal: string;
    if (pathKey === '*') {
      actualVal = r.bodyYAML;
    } else {
      let current: unknown = r.body;

      for (const part of parts) {
        if (
          current === null ||
          current === undefined ||
          typeof current !== 'object' ||
          Array.isArray(current)
        ) {
          return false;
        }
        current = (current as Record<string, unknown>)[part];
      }
      if (current === undefined || current === null) {
        return false;
      }
      if (typeof current === 'object') {
        return false;
      } else {
        actualVal = String(current);
      }
    }

    const patterns = Array.isArray(val) ? val : [val];
    return patterns.some((pat) => {
      try {
        const regex = new RegExp(String(pat), 'i');
        return regex.test(actualVal);
      } catch {
        return false;
      }
    });
  });
}
