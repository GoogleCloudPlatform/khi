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
 * View model representing the bottom-right search index progress overlay.
 */
export interface IndexProgressViewModel {
  /**
   * Whether the progress card should be displayed.
   */
  readonly visible: boolean;
  /**
   * Progress percentage (0 - 100).
   */
  readonly percent: number;
  /**
   * Detail message describing current indexing phase.
   */
  readonly message: string;
  /**
   * Whether the index is complete and ready.
   */
  readonly isReady: boolean;
}
