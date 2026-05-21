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

import {
  Event,
  TimelinePathNode,
  Revision,
  Timeline,
} from 'src/app/store/domain/timeline';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { InternedStructDecoder } from 'src/app/store/domain/struct-decoder';
import { InternedStruct } from 'src/app/generated/khifile/shared_pb';
import { RevisionState, TimelineType, Verb } from 'src/app/store/domain/style';
import { LogStore } from 'src/app/store/domain/log-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Raw timeline object interface from the assembler.
 */
export interface TimelineDTO {
  readonly id: number;
  readonly timelineTypeId: number;
  readonly nameStringId: number;
  readonly parentTimelineId: number;
  readonly revisionIds: readonly number[];
  readonly eventIds: readonly number[];
}

/**
 * Raw revision object interface from the assembler.
 */
export interface RevisionDTO {
  readonly id: number;
  readonly logId: number;
  readonly changedTime: bigint;
  readonly principalStringId: number;
  readonly verbTypeId: number;
  readonly stateTypeId: number;
  readonly body?: InternedStruct;
}

/**
 * Raw event object interface from the assembler.
 */
export interface EventDTO {
  readonly id: number;
  readonly logId: number;
}

/**
 * Store for managing and retrieving timelines, revisions, and events efficiently.
 */
export class TimelineStore {
  // Timeline metadata
  private timelineIds = new Uint32Array(0);
  private timelineTypeIds = new Uint32Array(0);
  private timelineNameStringIds = new Uint32Array(0);
  private timelineParentIds = new Uint32Array(0);
  private readonly timelineRevisionIds: Uint32Array[] = [];
  private readonly timelineEventIds: Uint32Array[] = [];
  private readonly timelineChildrenIds: number[][] = [];
  private readonly timelinesList: ReadonlyDomainElement<Timeline>[] = [];
  private timelineIdToIndex: { [tid: number]: number } = {};

  // Revision metadata stored in packed arrays
  private revisionIds = new Uint32Array(0);
  private revisionLogIds = new Uint32Array(0);
  private revisionChangedTimes = new BigUint64Array(0);
  private revisionPrincipalStringIds = new Uint32Array(0);
  private revisionVerbTypeIds = new Uint32Array(0);
  private revisionStateTypeIds = new Uint32Array(0);
  private readonly revisionBodies: InternedStruct[] = [];
  private revisionIdToIndex: { [rid: number]: number } = {};

  // Event metadata
  private eventIds = new Uint32Array(0);
  private eventLogIds = new Uint32Array(0);
  private eventIdToIndex: { [eid: number]: number } = {};

  private readonly decoder: InternedStructDecoder;

  constructor(
    private readonly internPool: InternPoolStore,
    private readonly styleStore: StyleStore,
    public readonly logStore: LogStore,
  ) {
    this.decoder = new InternedStructDecoder(this.internPool);
  }

  /**
   * Initializes the store with the raw timelines, revisions, and events.
   * @param timelines Iterable of raw timelines.
   * @param timelineCount Total number of timelines.
   * @param revisions Iterable of raw revisions.
   * @param revisionCount Total number of revisions.
   * @param events Iterable of raw events.
   * @param eventCount Total number of events.
   */
  public initialize(
    timelines: Iterable<TimelineDTO>,
    timelineCount: number,
    revisions: Iterable<RevisionDTO>,
    revisionCount: number,
    events: Iterable<EventDTO>,
    eventCount: number,
  ): void {
    this.timelineIds = new Uint32Array(timelineCount);
    this.timelineTypeIds = new Uint32Array(timelineCount);
    this.timelineNameStringIds = new Uint32Array(timelineCount);
    this.timelineParentIds = new Uint32Array(timelineCount);

    this.timelineRevisionIds.length = 0;
    this.timelineEventIds.length = 0;
    this.revisionBodies.length = 0;
    this.timelinesList.length = 0;
    this.timelineChildrenIds.length = 0;
    this.timelineIdToIndex = {};
    this.revisionIdToIndex = {};
    this.eventIdToIndex = {};

    // Load timelines
    let tIndex = 0;
    for (const t of timelines) {
      this.timelineIds[tIndex] = t.id;
      this.timelineTypeIds[tIndex] = t.timelineTypeId;
      this.timelineNameStringIds[tIndex] = t.nameStringId;
      this.timelineParentIds[tIndex] = t.parentTimelineId;

      this.timelineRevisionIds[tIndex] = new Uint32Array(t.revisionIds);
      this.timelineEventIds[tIndex] = new Uint32Array(t.eventIds);
      this.timelineChildrenIds[tIndex] = [];

      this.timelineIdToIndex[t.id] = tIndex;
      this.timelinesList.push(new Timeline(t.id, this));
      tIndex++;
    }

    // Build child timeline relationships
    for (const t of timelines) {
      if (t.parentTimelineId !== 0) {
        const pIndex = this.getTimelineIndex(t.parentTimelineId);
        this.timelineChildrenIds[pIndex].push(t.id);
      }
    }

    this.revisionIds = new Uint32Array(revisionCount);
    this.revisionLogIds = new Uint32Array(revisionCount);
    this.revisionChangedTimes = new BigUint64Array(revisionCount);
    this.revisionPrincipalStringIds = new Uint32Array(revisionCount);
    this.revisionVerbTypeIds = new Uint32Array(revisionCount);
    this.revisionStateTypeIds = new Uint32Array(revisionCount);

    // Load revisions
    let rIndex = 0;
    for (const r of revisions) {
      this.revisionIds[rIndex] = r.id;
      this.revisionLogIds[rIndex] = r.logId;
      this.revisionChangedTimes[rIndex] = r.changedTime;
      this.revisionPrincipalStringIds[rIndex] = r.principalStringId;
      this.revisionVerbTypeIds[rIndex] = r.verbTypeId;
      this.revisionStateTypeIds[rIndex] = r.stateTypeId;

      if (r.body !== undefined) {
        this.revisionBodies[rIndex] = r.body;
      }
      this.revisionIdToIndex[r.id] = rIndex;
      rIndex++;
    }

    // load events
    this.eventIds = new Uint32Array(eventCount);
    this.eventLogIds = new Uint32Array(eventCount);
    let eIndex = 0;
    for (const e of events) {
      this.eventIds[eIndex] = e.id;
      this.eventLogIds[eIndex] = e.logId;
      this.eventIdToIndex[e.id] = eIndex;
      eIndex++;
    }
  }

  // --- Timeline Accessors ---

  /**
   * Retrieves a specific timeline domain adapter by its ID.
   * @param id The unique timeline identifier.
   * @returns The readonly domain element adapter for the timeline.
   * @throws Error if the specified timeline ID is not found in the store.
   */
  public getTimeline(id: number): ReadonlyDomainElement<Timeline> {
    return this.timelinesList[this.getTimelineIndex(id)];
  }

  /**
   * Gets a readonly list of all timeline domain adapters in the store.
   */
  public get timelines(): readonly ReadonlyDomainElement<Timeline>[] {
    return this.timelinesList;
  }

  /**
   * Gets the name of a timeline by its ID.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getTimelineName(id: number): string {
    return this.internPool.getString(
      this.timelineNameStringIds[this.getTimelineIndex(id)],
    );
  }

  /**
   * Gets the categorization style classification of a timeline by its ID.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getTimelineType(id: number): ReadonlyDomainElement<TimelineType> {
    return this.styleStore.getTimelineType(
      this.timelineTypeIds[this.getTimelineIndex(id)],
    );
  }

  /**
   * Gets the parent timeline identification associated to the specified entity.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getTimelineParentId(id: number): number {
    return this.timelineParentIds[this.getTimelineIndex(id)];
  }

  /**
   * Evaluates nodes path from root timeline.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _computeTimelinePath(
    id: number,
  ): ReadonlyDomainElement<TimelinePathNode[]> {
    const path: TimelinePathNode[] = [];
    let currentId: number | null = id;

    while (currentId && currentId !== 0) {
      const idx = this.getTimelineIndex(currentId);
      path.push({
        id: currentId,
        type: this.styleStore.getTimelineType(this.timelineTypeIds[idx]),
        label: this.internPool.getString(this.timelineNameStringIds[idx]),
      });
      currentId = this.timelineParentIds[idx];
    }
    return path.reverse();
  }

  /**
   * Retrieves child revision adapters of a specific timeline.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getRevisionsForTimeline(
    id: number,
  ): ReadonlyDomainElement<Revision[]> {
    const revIds = this.timelineRevisionIds[this.getTimelineIndex(id)];
    if (!revIds) {
      return [];
    }

    const revisions: Revision[] = [];
    for (let i = 0; i < revIds.length; i++) {
      revisions.push(new Revision(revIds[i], id, this, i));
    }
    revisions.sort((r1, r2) => Number(r1.changedTime - r2.changedTime));
    return revisions;
  }

  /**
   * Retrieves child events of a specific timeline.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getEventsForTimeline(id: number): ReadonlyDomainElement<Event[]> {
    const eventIds = this.timelineEventIds[this.getTimelineIndex(id)];
    if (!eventIds) {
      return [];
    }

    const events: Event[] = [];
    for (let i = 0; i < eventIds.length; i++) {
      events.push(new Event(eventIds[i], id, this));
    }
    events.sort((e1, e2) => e1.logIndex - e2.logIndex);
    return events;
  }

  /**
   * Retrieves child timeline ID references for a specific timeline.
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _getChildIdsForTimeline(id: number): readonly number[] {
    return this.timelineChildrenIds[this.getTimelineIndex(id)] ?? [];
  }

  private getTimelineIndex(id: number): number {
    const index = this.timelineIdToIndex[id];
    if (index === undefined) {
      throw new Error(`Timeline ID ${id} not found`);
    }
    return index;
  }

  // --- Revision Accessors ---

  /**
   * Gets state timestamp evaluation for a revision by its ID.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionChangedTime(id: number): bigint {
    return this.revisionChangedTimes[this.getRevisionIndex(id)];
  }

  /**
   * Gets state principal execution user string for a revision by its ID.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionPrincipal(id: number): string {
    return this.internPool.getString(
      this.revisionPrincipalStringIds[this.getRevisionIndex(id)],
    );
  }

  /**
   * Gets revision verb execution data from store by its ID.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionVerb(id: number): ReadonlyDomainElement<Verb> {
    return this.styleStore.getVerb(
      this.revisionVerbTypeIds[this.getRevisionIndex(id)],
    );
  }

  /**
   * Gets revision presentation state categorization details by its ID.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionState(id: number): ReadonlyDomainElement<RevisionState> {
    return this.styleStore.getRevisionState(
      this.revisionStateTypeIds[this.getRevisionIndex(id)],
    );
  }

  /**
   * Gets revision source log element ID reference.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionLogId(id: number): number {
    return this.revisionLogIds[this.getRevisionIndex(id)];
  }

  /**
   * Decodes encapsulated revision properties from its structural domain interface.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _decodeRevisionBody(
    id: number,
  ): ReadonlyDomainElement<Record<string, unknown>> | null {
    const struct = this.revisionBodies[this.getRevisionIndex(id)];
    if (!struct) return null;
    return this.decoder.decode(struct);
  }

  private getRevisionIndex(id: number): number {
    const index = this.revisionIdToIndex[id];
    if (index === undefined) {
      throw new Error(`Revision ID ${id} not found`);
    }
    return index;
  }

  // --- Event Accessors ---

  /**
   * Gets underlying data log ID for resource event by its ID.
   * @note Intended solely for internal retrieval inside the {@link Event} domain adapter.
   */
  public _getEventLogId(id: number): number {
    return this.eventLogIds[this.getEventIndex(id)];
  }

  private getEventIndex(id: number): number {
    const index = this.eventIdToIndex[id];
    if (index === undefined) {
      throw new Error(`Event ID ${id} not found`);
    }
    return index;
  }
}
