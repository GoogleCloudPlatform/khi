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

import { Provider, SkipSelf } from '@angular/core';

/**
 * @fileoverview Abstract class and implementations for event reporting.
 */

/**
 * Abstract class for event reporting.
 * Used as a DI token in Angular.
 */
export abstract class Reporter {
  /**
   * Creates an instance of Reporter.
   * @param labels Static labels to add to every event.
   * @param parent Optional parent reporter.
   */
  constructor(
    protected labels: Record<string, unknown>,
    protected readonly parent: Reporter | null,
  ) {}

  /**
   * Sends the event data.
   * @param event The event data to send.
   */
  abstract send(event: Record<string, unknown>): void;

  /**
   * Adds labels to be included with future log calls.
   * @param labels The labels to add.
   */
  addLabels(labels: Record<string, unknown>): void {
    this.labels = { ...this.labels, ...labels };
  }
}

/**
 * An implementation of Reporter that logs events to the console.
 */
export class ConsoleReporter extends Reporter {
  /**
   * Creates an instance of ConsoleReporter.
   */
  constructor() {
    super({}, null);
  }

  /**
   * Logs the event data to the console.
   * @param event The event data to log.
   */
  override send(event: Record<string, unknown>): void {
    const merged = { ...this.labels, ...event };
    console.log('Event Reported:', merged);
  }
}

/**
 * Reporter that delegates to a parent reporter while adding its own static labels.
 */
export class HierarchicalReporter extends Reporter {
  /**
   * Merges event data with static labels and delegates to parent.
   * Throws error if no parent is found.
   * @param event The event data to send.
   */
  override send(event: Record<string, unknown>): void {
    const merged = { ...this.labels, ...event };
    if (!this.parent) {
      throw new Error('No parent Reporter found');
    }
    this.parent.send(merged);
  }
}

/**
 * Provides labels at a component or module level.
 * It overrides the `Reporter` provider to use `HierarchicalReporter`.
 *
 * @param labels Static labels to apply in this context.
 * @returns A single provider.
 */
export function provideReporterContext(
  labels: Record<string, unknown>,
): Provider {
  return {
    provide: Reporter,
    useFactory: (parent: Reporter) => {
      return new HierarchicalReporter(labels, parent);
    },
    deps: [[new SkipSelf(), Reporter]],
  };
}
