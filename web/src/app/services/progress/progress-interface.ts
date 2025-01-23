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

import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';

/**
 * CurrentProgress is a data type to represent the current status of a task.
 */
export interface CurrentProgress {
  /**
   * A message string shown at the bottom of the progress bar.
   */
  message: string;
  /**
   * Percentage of the progress bar.
   * This value is disregarded when mode field is `indeterminate`.
   */
  percent: number;
  /**
   * The mode of progress bar.
   */
  mode: 'indeterminate' | 'determinate';
}

/**
 * Angular injection token to receive ProgressDialogStatusObserver.
 */
export const PROGRESS_DIALOG_STATUS_OBSERVER =
  new InjectionToken<ProgressDialogStatusObserver>(
    'PROGRESS_DIALOG_STATUS_OBSERVER',
  );

/**
 * Angular injection token to receive ProgressDialogStatusUpdator.
 */
export const PROGRESS_DIALOG_STATUS_UPDATOR =
  new InjectionToken<ProgressDialogStatusUpdator>(
    'PROGRESS_DIALOG_STATUS_UPDATOR',
  );

/**
 * A interface to receive the current progress status of application.
 */
export interface ProgressDialogStatusObserver {
  /**
   * status returns the Observable contains the state.
   */
  status(): Observable<CurrentProgress>;
}

/**
 * A interface to update the current progress status of application.
 */
export interface ProgressDialogStatusUpdator {
  /**
   * Show the progress dialog.
   */
  show(): void;
  /**
   * Dismiss the progress dialog.
   */
  dismiss(): void;

  /**
   * Setting the progress of dialog.
   */
  updateProgress(progress: CurrentProgress): void;
}
