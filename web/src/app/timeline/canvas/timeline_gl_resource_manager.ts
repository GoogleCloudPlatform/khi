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

import { filter, map, Observable, Subject } from 'rxjs';

export class LRULifetimeManager<T> {
  private readonly cache: Map<T, void> = new Map<T, void>();
  constructor(public readonly capacity: number) {}

  public touch(data: T): T | null {
    if (this.cache.has(data)) {
      this.cache.delete(data);
    }
    this.cache.set(data, void 0);
    if (this.cache.size > this.capacity) {
      // Map implementation after ES2015 should return the oldest item at first.
      for (const item of this.cache.keys()) {
        this.cache.delete(item);
        return item;
      }
    }
    return null;
  }
}
/**
 * TimelineGLResourceManager monitor how many timelines have loaded gl resources on GPU and notify timeline to release resources on GPU.
 */
export class TimelineGLResourceManager {
  /**
   * The maximum count of timelines to be kept on GPU.
   */
  public static readonly MAX_TIMELINE_COUNT_ON_GPU = 500;

  private readonly timelineLru: LRULifetimeManager<string> =
    new LRULifetimeManager<string>(
      TimelineGLResourceManager.MAX_TIMELINE_COUNT_ON_GPU,
    );

  /**
   * emits the timeline ID to be released from GPU.
   */
  private readonly unloadTimelineIdSubject = new Subject<string>();

  public onUnload(timelinePath: string): Observable<void> {
    return this.unloadTimelineIdSubject.pipe(
      filter((id) => id === timelinePath),
      map(() => void 0),
    );
  }

  /**
   * Mark a specific timeline path used.
   * @param timelinePath
   */
  public touch(timelinePath: string): void {
    const released = this.timelineLru.touch(timelinePath);
    if (released !== null) {
      this.unloadTimelineIdSubject.next(released);
    }
  }
}
