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
 * Result of parsing a KHI job mode CLI command.
 */
export interface ParsedJobCommand {
  /**
   * The inspection type identifier.
   */
  readonly inspectionType: string;

  /**
   * List of enabled feature identifiers.
   */
  readonly features: readonly string[];

  /**
   * Key-value parameters passed to the inspection.
   */
  readonly parameters: Record<string, unknown>;
}

/**
 * Extracts flag values from a bash command line string.
 * Supports single quotes, double quotes, line-continuation backslashes, and bash quote concatenation.
 *
 * @param command The raw command line string.
 * @returns Map of flag names to their parsed string values.
 */
export function extractFlagsFromCommand(command: string): Map<string, string> {
  const flags = new Map<string, string>();
  // Remove backslash line continuations
  const normalized = command.replace(/\\\r?\n/g, ' ');

  let i = 0;
  const len = normalized.length;

  while (i < len) {
    // Skip whitespace
    while (i < len && /\s/.test(normalized[i])) {
      i++;
    }
    if (i >= len) {
      break;
    }

    // Check if this token starts with --
    if (normalized[i] === '-' && i + 1 < len && normalized[i + 1] === '-') {
      i += 2;
      // Read flag name until '=', space, or end of string
      const nameStart = i;
      while (i < len && normalized[i] !== '=' && !/\s/.test(normalized[i])) {
        i++;
      }
      const flagName = normalized.slice(nameStart, i);

      let flagValue = '';
      if (i < len && normalized[i] === '=') {
        i++; // skip '='
        flagValue = readArgumentValue(normalized, i, (newIndex) => {
          i = newIndex;
        });
      } else if (i < len && /\s/.test(normalized[i])) {
        // Lookahead to see if next token is a value or another flag
        let j = i;
        while (j < len && /\s/.test(normalized[j])) {
          j++;
        }
        if (j < len && !normalized.startsWith('--', j)) {
          i = j;
          flagValue = readArgumentValue(normalized, i, (newIndex) => {
            i = newIndex;
          });
        }
      }

      flags.set(flagName, flagValue);
    } else {
      // Non-flag token, skip over it
      readArgumentValue(normalized, i, (newIndex) => {
        i = newIndex;
      });
    }
  }

  return flags;
}

/**
 * Reads a single argument value starting at the given index, handling quotes and escapes.
 */
function readArgumentValue(
  input: string,
  startIndex: number,
  setIndex: (index: number) => void,
): string {
  let result = '';
  let i = startIndex;
  const len = input.length;

  while (i < len && !/\s/.test(input[i])) {
    if (input[i] === "'") {
      i++; // skip opening single quote
      while (i < len) {
        if (input[i] === "'") {
          // Check for bash concatenation with escaped single quote: '\''
          if (
            i + 3 < len &&
            input[i + 1] === '\\' &&
            input[i + 2] === "'" &&
            input[i + 3] === "'"
          ) {
            result += "'";
            i += 4;
          } else {
            i++; // skip closing single quote
            break;
          }
        } else {
          result += input[i];
          i++;
        }
      }
    } else if (input[i] === '"') {
      i++; // skip opening double quote
      while (i < len) {
        if (input[i] === '\\' && i + 1 < len) {
          result += input[i + 1];
          i += 2;
        } else if (input[i] === '"') {
          i++; // skip closing double quote
          break;
        } else {
          result += input[i];
          i++;
        }
      }
    } else if (input[i] === '\\' && i + 1 < len) {
      result += input[i + 1];
      i += 2;
    } else {
      result += input[i];
      i++;
    }
  }

  setIndex(i);
  return result;
}

/**
 * Parses a KHI Job Mode CLI command string into inspection parameters.
 *
 * @param command The job mode command line string.
 * @returns ParsedJobCommand containing inspectionType, features, and parameters.
 * @throws Error if command string is empty, missing required flags, or contains invalid JSON.
 */
export function parseJobModeCommand(command: string): ParsedJobCommand {
  const trimmed = command.trim();
  if (!trimmed) {
    throw new Error('Command is empty.');
  }

  const flags = extractFlagsFromCommand(trimmed);

  const inspectionType = flags.get('job-inspection-type');
  if (!inspectionType) {
    throw new Error('Missing required flag: --job-inspection-type');
  }

  const featuresRaw = flags.get('job-inspection-features') ?? '';
  const features = featuresRaw
    .split(',')
    .map((f) => f.trim())
    .filter((f) => f.length > 0);

  const valuesRaw = flags.get('job-inspection-values');
  let parameters: Record<string, unknown> = {};
  if (valuesRaw && valuesRaw.trim().length > 0) {
    try {
      const parsed = JSON.parse(valuesRaw.trim());
      if (
        typeof parsed !== 'object' ||
        parsed === null ||
        Array.isArray(parsed)
      ) {
        throw new Error('Expected JSON object.');
      }
      parameters = parsed as Record<string, unknown>;
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      throw new Error(`Failed to parse --job-inspection-values JSON: ${msg}`);
    }
  }

  return {
    inspectionType,
    features,
    parameters,
  };
}
