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
 * Formats a byte count into a human-readable string with binary units (B, KB, MB, GB, TB).
 *
 * @param bytes - Size in bytes.
 * @returns Human-readable formatted string (e.g., "512 B", "1.0 KB", "10 MB").
 */
export function formatBytes(bytes: number): string {
  if (bytes <= 0 || !Number.isFinite(bytes)) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const digitGroup = Math.max(
    0,
    Math.min(
      Math.floor(Math.log10(bytes) / Math.log10(1024)),
      units.length - 1,
    ),
  );
  const value = bytes / Math.pow(1024, digitGroup);
  return `${value.toFixed(value >= 10 || digitGroup === 0 ? 0 : 1)} ${units[digitGroup]}`;
}
