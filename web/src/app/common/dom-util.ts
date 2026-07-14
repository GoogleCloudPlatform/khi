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
 * Checks if the current platform is macOS.
 * @returns true if the platform is macOS.
 */
export function isMac(): boolean {
  if (typeof navigator === 'undefined') {
    return false;
  }
  return (
    navigator.userAgent.includes('Mac OS X') ||
    navigator.userAgent.includes('Macintosh')
  );
}

/**
 * Checks if the given keyboard event is a standard search shortcut.
 * Uses Cmd+F on macOS and Ctrl+F on Windows/Linux.
 * @param event The keyboard event.
 * @returns true if it is the platform's search shortcut.
 */
export function isSearchShortcut(event: KeyboardEvent): boolean {
  if (event.key.toLowerCase() !== 'f') {
    return false;
  }
  if (isMac()) {
    return event.metaKey && !event.ctrlKey && !event.altKey && !event.shiftKey;
  }
  return event.ctrlKey && !event.metaKey && !event.altKey && !event.shiftKey;
}

/**
 * Checks if the event is originating from a dialog or overlay (e.g., Material Dialog).
 * @param event The keyboard event.
 * @returns true if the event originated from an overlay.
 */
export function isEventFromOverlay(event: Event): boolean {
  const target = event.target as Element | null;
  return !!target?.closest?.('.cdk-overlay-pane');
}
