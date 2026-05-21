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
    public readonly index: number,
  ) {}

  /**
   * Gets the next revision in chronological order on this timeline, or null if this is the latest.
   */
  get next(): ReadonlyDomainElement<Revision> | null {
    const revisions = this.timeline.revisions;
    return this.index < revisions.length - 1 ? revisions[this.index + 1] : null;
  }

  /**
   * Gets the previous revision in chronological order on this timeline, or null if this is the first.
   */
  get prev(): ReadonlyDomainElement<Revision> | null {
    const revisions = this.timeline.revisions;
    return this.index > 0 ? revisions[this.index - 1] : null;
  }

  /**
   * Gets the end time of this revision in nanoseconds.
   * If there is a subsequent revision, returns the start time of that revision.
   * Otherwise, returns null.
   */
  public getEndNs(): bigint | null {
    const n = this.next;
    return n ? n.changedTime : null;
  }

  /**
   * Gets the end time of this revision in milliseconds.
   * @deprecated Use {@link getEndNs} instead.
   */
  public legacyGetEndMs(): number | null {
    const n = this.next;
    return n ? n.legacyChangedTimeMs : null;
  }

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
}
