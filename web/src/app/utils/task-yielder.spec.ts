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

import { TaskYielder } from './task-yielder';

describe('TaskYielder', () => {
  it('should not yield if elapsed time does not exceed maxProcessingTimeMs', async () => {
    const yielder = new TaskYielder(10000);
    const startTime = performance.now();
    await yielder.yield();
    const duration = performance.now() - startTime;
    expect(duration).toBeLessThan(50);
  });

  it('should yield after maxProcessingTimeMs exceeded', async () => {
    const yielder = new TaskYielder(0);
    await new Promise((resolve) => setTimeout(resolve, 5));
    await expectAsync(yielder.yield()).toBeResolved();
  });

  it('should throw DOMException when aborted initially', async () => {
    const controller = new AbortController();
    controller.abort();
    const yielder = new TaskYielder(16, controller.signal);

    await expectAsync(yielder.yield()).toBeRejectedWithError(DOMException);
  });
});
