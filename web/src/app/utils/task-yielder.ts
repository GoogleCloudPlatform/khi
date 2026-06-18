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
 * Represents an error thrown when an operation is cancelled or aborted.
 */
export class CancellationError extends Error {
  /**
   * Initializes a new CancellationError.
   * @param message The optional error message.
   */
  constructor(message = 'The operation was cancelled') {
    super(message);
    this.name = 'CancellationError';
  }
}

/**
 * Utility class to cooperatively yield execution in heavy synchronous loops
 * to avoid UI freezing.
 */
export class TaskYielder {
  private lastYieldTime: number;

  /**
   * Initializes a new TaskYielder.
   * @param maxProcessingTimeMs The maximum processing time per chunk in milliseconds.
   * @param abortSignal The optional signal to cancel processing.
   */
  constructor(
    private readonly maxProcessingTimeMs: number = 16,
    private readonly abortSignal?: AbortSignal,
  ) {
    this.lastYieldTime = performance.now();
  }

  /**
   * Yields execution if the elapsed time since the last yield exceeds maxProcessingTimeMs.
   * Throws CancellationError if the abortSignal is aborted.
   */
  public async yield(): Promise<void> {
    if (this.abortSignal?.aborted) {
      throw new CancellationError();
    }
    if (performance.now() - this.lastYieldTime > this.maxProcessingTimeMs) {
      await new Promise<void>((resolve) => setTimeout(resolve, 0));
      if (this.abortSignal?.aborted) {
        throw new CancellationError();
      }
      this.lastYieldTime = performance.now();
    }
  }

  /**
   * Resets the internal yield timer to the current time.
   */
  public reset(): void {
    this.lastYieldTime = performance.now();
  }
}
