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

import { RevisionState, TimelineType, Verb } from 'src/app/store/domain/style';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import * as yaml from 'js-yaml';

import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';
import { bisectLeft } from 'src/app/common/misc-util';

/**
 * Represents a node in the path from the root timeline.
 */
export interface TimelinePathNode {
  readonly id: number;
  readonly type: TimelineType;
  readonly label: string;
}

/**
 * Lazy adapter for a resource revision.
 */
export class Revision {
  private _body?: ReadonlyDomainElement<Record<string, unknown>> | null;
  private _log?: ReadonlyDomainElement<Log>;

  constructor(
    public readonly id: number,
    public readonly timelineId: number,
    private readonly timelineStore: TimelineStore,
  ) {}

  /**
   * Gets the timeline this revision belongs to.
   */
  get timeline(): ReadonlyDomainElement<Timeline> {
    return this.timelineStore.getTimeline(this.timelineId);
  }

  /**
   * Gets the Unix timestamp (in nanoseconds) when the state changed.
   */
  get changedTime(): bigint {
    return this.timelineStore._getRevisionChangedTime(this.id);
  }

  /**
   * Gets the Unix timestamp (in milliseconds) when the state changed.
   * @deprecated Use {@link changedTime} instead, which returns the timestamp in nanoseconds.
   */
  get legacyChangedTimeMs(): number {
    return BigIntTimeUtil.NsToNumberMs(this.changedTime);
  }

  /**
   * Gets the user identity or agent who initiated the state change.
   */
  get principal(): string {
    return this.timelineStore._getRevisionPrincipal(this.id);
  }

  /**
   * Gets the associated audit verb metadata.
   */
  get verb(): ReadonlyDomainElement<Verb> {
    return this.timelineStore._getRevisionVerb(this.id);
  }

  /**
   * Gets the visual state presentation category metadata.
   */
  get state(): ReadonlyDomainElement<RevisionState> {
    return this.timelineStore._getRevisionState(this.id);
  }

  /**
   * Gets the underlying log record for the specific state revision.
   */
  get log(): ReadonlyDomainElement<Log> {
    if (!this._log) {
      this._log = this.timelineStore.logStore.getLog(
        this.timelineStore._getRevisionLogId(this.id),
      );
    }
    return this._log;
  }

  /**
   * Gets the optional structured resource manifest parameters at the snapshot moment.
   */
  get body(): ReadonlyDomainElement<Record<string, unknown>> | null {
    if (this._body === undefined) {
      this._body = this.timelineStore._decodeRevisionBody(this.id);
    }
    return this._body;
  }

  /**
   * Gets the YAML string representation of the resource manifest.
   */
  get bodyYAML(): string {
    return this.body ? yaml.dump(this.body, { lineWidth: -1 }) : '';
  }

  /**
   * Gets the chronological index of the underlying log.
   */
  get logIndex(): number {
    return this.log.logIndex;
  }
}

/**
 * Lazy adapter for an event on a timeline.
 */
export class Event {
  private _log?: ReadonlyDomainElement<Log>;

  constructor(
    public readonly id: number,
    public readonly timelineId: number,
    private readonly timelineStore: TimelineStore,
  ) {}

  /**
   * Gets the timeline this event belongs to.
   */
  get timeline(): ReadonlyDomainElement<Timeline> {
    return this.timelineStore.getTimeline(this.timelineId);
  }

  /**
   * Gets the underlying log record for this specific resource event.
   */
  get log(): ReadonlyDomainElement<Log> {
    if (!this._log) {
      this._log = this.timelineStore.logStore.getLog(
        this.timelineStore._getEventLogId(this.id),
      );
    }
    return this._log;
  }

  /**
   * Gets the chronological index of the underlying log.
   */
  get logIndex(): number {
    return this.log.logIndex;
  }

  /**
   * Gets the timestamp of the log in nanoseconds.
   */
  get timestamp(): bigint {
    return this.log.timestamp;
  }

  /**
   * Gets the timestamp in milliseconds.
   * @deprecated Use log.timestamp.
   */
  get legacyTimestamp(): number {
    return this.log.legacyTimestampMs;
  }
}

/**
 * Lazy adapter for a timeline.
 */
export class Timeline {
  private _path?: ReadonlyDomainElement<TimelinePathNode[]>;
  private _revisions?: ReadonlyDomainElement<Revision[]>;
  private _events?: ReadonlyDomainElement<Event[]>;

  constructor(
    public readonly id: number,
    private readonly timelineStore: TimelineStore,
  ) {}

  /**
   * Gets the localized string name label for the timeline.
   */
  get name(): string {
    return this.timelineStore._getTimelineName(this.id);
  }

  /**
   * Gets the classification presentation styling applied to this timeline tracking instance.
   */
  get type(): ReadonlyDomainElement<TimelineType> {
    return this.timelineStore._getTimelineType(this.id);
  }

  /**
   * Gets the calculated structural path node trace.
   */
  get path(): ReadonlyDomainElement<TimelinePathNode[]> {
    if (this._path === undefined) {
      this._path = this.timelineStore._computeTimelinePath(this.id);
    }
    return this._path;
  }

  /**
   * Returns the debug representation of the timeline's path from its parent.
   */
  get debugPathText(): string {
    return this.path.map((n) => n.label).join('/');
  }

  /**
   * Gets the depth layer of this timeline in the hierarchy.
   * Returns 0 if the timeline has no parent, 1 if there is one parent, and so on.
   */
  get layer(): number {
    return this.path.length - 1;
  }

  /**
   * Gets the sequence array of status changes associated with this timeline.
   */
  get revisions(): ReadonlyDomainElement<Revision[]> {
    if (this._revisions === undefined) {
      this._revisions = this.timelineStore._getRevisionsForTimeline(this.id);
    }
    return this._revisions;
  }

  /**
   * Gets the list events associated with this timeline tracking stream.
   */
  get events(): ReadonlyDomainElement<Event[]> {
    if (this._events === undefined) {
      this._events = this.timelineStore._getEventsForTimeline(this.id);
    }
    return this._events;
  }

  /**
   * Retrieves the parent timeline adapter, or null if this is a root timeline.
   */
  parent(): ReadonlyDomainElement<Timeline> | null {
    const pId = this.timelineStore._getTimelineParentId(this.id);
    return pId === 0 ? null : this.timelineStore.getTimeline(pId);
  }

  /**
   * Gets the number of child timelines associated with this timeline.
   */
  get childrenCount(): number {
    return this.timelineStore._getChildIdsForTimeline(this.id).length;
  }

  /**
   * Iterates over the child timelines associated with this timeline.
   */
  *children(): IterableIterator<ReadonlyDomainElement<Timeline>> {
    const childIds = this.timelineStore._getChildIdsForTimeline(this.id);
    for (let i = 0; i < childIds.length; i++) {
      yield this.timelineStore.getTimeline(childIds[i]);
    }
  }

  /**
   * Gets all descendant child timelines recursively.
   */
  *descendants(): IterableIterator<ReadonlyDomainElement<Timeline>> {
    for (const child of this.children()) {
      yield child;
      yield* child.descendants();
    }
  }

  /**
   * Get a pair of revisions from given log index.
   * @param logIndex
   */
  public getRevisionPairByLogId(logIndex: number): {
    readonly previous: ReadonlyDomainElement<Revision> | null;
    readonly current: ReadonlyDomainElement<Revision>;
  } | null {
    if (this.revisions.length === 0) return null;
    for (let i = this.revisions.length - 1; i >= 0; i--) {
      if (this.revisions[i].logIndex === logIndex) {
        if (i !== 0) {
          return {
            previous: this.revisions[i - 1],
            current: this.revisions[i],
          };
        }
        return { previous: null, current: this.revisions[i] };
      }
    }
    console.warn(
      `Attempted to find the logIndex ${logIndex} from timeline ${this.path.map((n) => n.label).join('/')}, but not found.`,
    );
    return null;
  }

  /**
   * Looks up an event on this timeline by its associated log using binary search.
   * Returns the event if found, otherwise null.
   */
  public lookupEventFromLog(
    log: ReadonlyDomainElement<Log> | null,
  ): ReadonlyDomainElement<Event> | null {
    if (!log) return null;
    const arr = this.events;
    const idx = bisectLeft(
      arr,
      log.logIndex,
      (item, target) => item.logIndex - target,
    );
    if (idx < arr.length && arr[idx].logIndex === log.logIndex) {
      return arr[idx];
    }
    return null;
  }

  /**
   * Looks up a revision on this timeline by its associated log using binary search.
   * Returns the revision if found, otherwise null.
   */
  public lookupRevisionFromLog(
    log: ReadonlyDomainElement<Log> | null,
  ): ReadonlyDomainElement<Revision> | null {
    if (!log) return null;
    const arr = this.revisions;
    const idx = bisectLeft(
      arr,
      log.logIndex,
      (item, target) => item.logIndex - target,
    );
    if (idx < arr.length && arr[idx].logIndex === log.logIndex) {
      return arr[idx];
    }
    return null;
  }
}
