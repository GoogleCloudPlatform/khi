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

import { Injectable, inject } from '@angular/core';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';

/**
 * Result of a CEL expression validation query.
 */
export interface CelValidationResult {
  /** Whether the CEL expression syntax and variable types are valid. */
  readonly valid: boolean;
  /** Error message describing compilation or syntax failures if invalid. */
  readonly errorMessage: string;
}

/**
 * CelValidationClientService provides methods to validate CEL expressions against backend schemas.
 */
@Injectable({
  providedIn: 'root',
})
export class CelValidationClientService {
  private readonly connectClientService = inject(ConnectClientService);

  /**
   * Validates a timeline CEL query expression against registered schemas and custom functions.
   *
   * @param query - The CEL query string to validate.
   * @param signal - Optional AbortSignal to cancel the RPC request.
   * @returns The validation result indicating validity and error details.
   */
  public async validateTimelineQuery(
    query: string,
    signal?: AbortSignal,
  ): Promise<CelValidationResult> {
    if (query.trim() === '') {
      return { valid: true, errorMessage: '' };
    }
    try {
      const res =
        await this.connectClientService.celValidationClient.validateTimelineQuery(
          { query },
          { signal },
        );
      return {
        valid: res.valid,
        errorMessage: res.errorMessage,
      };
    } catch (err) {
      if (signal?.aborted) {
        return { valid: true, errorMessage: '' };
      }
      return {
        valid: false,
        errorMessage: err instanceof Error ? err.message : String(err),
      };
    }
  }

  /**
   * Validates a log CEL query expression against registered schemas and custom functions.
   *
   * @param query - The CEL query string to validate.
   * @param signal - Optional AbortSignal to cancel the RPC request.
   * @returns The validation result indicating validity and error details.
   */
  public async validateLogQuery(
    query: string,
    signal?: AbortSignal,
  ): Promise<CelValidationResult> {
    if (query.trim() === '') {
      return { valid: true, errorMessage: '' };
    }
    try {
      const res =
        await this.connectClientService.celValidationClient.validateLogQuery(
          { query },
          { signal },
        );
      return {
        valid: res.valid,
        errorMessage: res.errorMessage,
      };
    } catch (err) {
      if (signal?.aborted) {
        return { valid: true, errorMessage: '' };
      }
      return {
        valid: false,
        errorMessage: err instanceof Error ? err.message : String(err),
      };
    }
  }
}
