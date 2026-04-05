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

import { ErrorHandler, Injectable, inject } from '@angular/core';
import { Reporter } from './reporter';

/**
 * Reports unhandled errors to the Reporter.
 */
@Injectable()
export class ReporterErrorHandler implements ErrorHandler {
  private reporter = inject(Reporter);

  /**
   * Handles the error and reports it.
   * @param error The error to handle.
   */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  handleError(error: any): void {
    const message = error?.message || String(error);
    const stack = error?.stack || '';

    this.reporter.send({
      event: 'unhandled_error',
      message: message,
      stack: stack,
    });

    // Delegates to console.error to maintain default behavior (logging to console).
    console.error(error);
  }
}
