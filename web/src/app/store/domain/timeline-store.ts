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
import {
  RevisionState,
  TimelineType,
  Verb,
  StyleProvider,
} from 'src/app/store/domain/style';
import { align } from 'src/app/common/memory-util';
import { LogStore } from 'src/app/store/domain/log-store';
import {
  DomainFieldAnnotation,
  ReadonlyDomainElement,
  allocateBuffer,
} from 'src/app/store/domain/types';

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

export interface FieldAnnotationDTO {
  readonly fieldPathStringId: number;
  readonly mutatingWebhook?: {
    readonly configurationStringId: number;
    readonly webhookStringId: number;
    readonly round: number;
    readonly index: number;
  };
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
  readonly resourceBodyStructId?: number;
  readonly fieldAnnotations?: readonly FieldAnnotationDTO[];
}

/**
 * Raw event object interface from the assembler.
 */
export interface EventDTO {
  readonly id: number;
  readonly logId: number;
}

interface TimelineStoreLayout {
  timelineIds: number;
  timelineTypeIds: number;
  timelineNameStringIds: number;
  timelineParentIds: number;
  revisionIds: number;
  revisionLogIds: number;
  revisionChangedTimes: number;
  revisionPrincipalStringIds: number;
  revisionVerbTypeIds: number;
  revisionStateTypeIds: number;
  revisionBodyStructIds: number;
  eventIds: number;
  eventLogIds: number;
  totalBytes: number;
}

/**
 * Store for managing and retrieving timelines, revisions, and events efficiently.
 */
export class TimelineStore {
  private metadataBuffer!: ArrayBuffer;

  // Timeline views
  private timelineIds!: Uint32Array;
  private timelineTypeIds!: Uint32Array;
  private timelineNameStringIds!: Uint32Array;
  private timelineParentIds!: Uint32Array;

  private readonly timelineRevisionIds: Uint32Array[] = [];
  private readonly timelineEventIds: Uint32Array[] = [];
  private readonly timelineChildrenIds: number[][] = [];
  private readonly timelinesList: ReadonlyDomainElement<Timeline>[] = [];
  private timelineIdToIndex: { [tid: number]: number } = {};

  // Revision views
  private revisionIds!: Uint32Array;
  private revisionLogIds!: Uint32Array;
  private revisionChangedTimes!: BigUint64Array;
  private revisionPrincipalStringIds!: Uint32Array;
  private revisionVerbTypeIds!: Uint32Array;
  private revisionStateTypeIds!: Uint32Array;
  private revisionBodyStructIds!: Uint32Array;
  private revisionFieldAnnotations: (
    readonly FieldAnnotationDTO[] | undefined
  )[] = [];
  private revisionIdToIndex: { [rid: number]: number } = {};

  // Event views
  private eventIds!: Uint32Array;
  private eventLogIds!: Uint32Array;
  private eventIdToIndex: { [eid: number]: number } = {};

  private constructor(
    private readonly internPool: InternPoolStore,
    public readonly styleStore: StyleProvider,
    public readonly logStore: LogStore,
  ) {}

  /**
   * Initializes a new TimelineStore instance with the raw timelines, revisions, and events.
   *
   * @param internPool Intern pool store for string interning.
   * @param styleStore Style provider for styling information.
   * @param logStore Log store providing log references.
   * @param timelines Iterable of raw timelines.
   * @param timelineCount Total number of timelines.
   * @param revisions Iterable of raw revisions.
   * @param revisionCount Total number of revisions.
   * @param events Iterable of raw events.
   * @param eventCount Total number of events.
   * @returns An initialized TimelineStore instance.
   */
  public static initialize(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    logStore: LogStore,
    timelines: Iterable<TimelineDTO>,
    timelineCount: number,
    revisions: Iterable<RevisionDTO>,
    revisionCount: number,
    events: Iterable<EventDTO>,
    eventCount: number,
  ): TimelineStore {
    const store = new TimelineStore(internPool, styleStore, logStore);
    store.load(
      timelines,
      timelineCount,
      revisions,
      revisionCount,
      events,
      eventCount,
    );
    return store;
  }

  private allocateMetadata(
    timelineCapacity: number,
    revisionCapacity: number,
    eventCapacity: number,
  ): void {
    const layout = this.calculateOffsets(
      timelineCapacity,
      revisionCapacity,
      eventCapacity,
    );
    this.metadataBuffer = allocateBuffer(layout.totalBytes);
    this.applyViews(layout, timelineCapacity, revisionCapacity, eventCapacity);
  }

  private calculateOffsets(
    tCap: number,
    rCap: number,
    eCap: number,
  ): TimelineStoreLayout {
    let offset = 0;

    const timelineIds = offset;
    offset += tCap * 4;

    const timelineTypeIds = offset;
    offset += tCap * 4;

    const timelineNameStringIds = offset;
    offset += tCap * 4;

    const timelineParentIds = offset;
    offset += tCap * 4;

    const revisionIds = align(offset, 4);
    offset = revisionIds + rCap * 4;

    const revisionLogIds = offset;
    offset += rCap * 4;

    const revisionPrincipalStringIds = offset;
    offset += rCap * 4;

    const revisionVerbTypeIds = offset;
    offset += rCap * 4;

    const revisionStateTypeIds = offset;
    offset += rCap * 4;

    const revisionBodyStructIds = offset;
    offset += rCap * 4;

    const revisionChangedTimes = align(offset, 8);
    offset = revisionChangedTimes + rCap * 8;

    const eventIds = align(offset, 4);
    offset = eventIds + eCap * 4;

    const eventLogIds = offset;
    offset += eCap * 4;

    return {
      timelineIds,
      timelineTypeIds,
      timelineNameStringIds,
      timelineParentIds,
      revisionIds,
      revisionLogIds,
      revisionPrincipalStringIds,
      revisionVerbTypeIds,
      revisionStateTypeIds,
      revisionBodyStructIds,
      revisionChangedTimes,
      eventIds,
      eventLogIds,
      totalBytes: offset,
    };
  }

  private applyViews(
    layout: TimelineStoreLayout,
    tCap: number,
    rCap: number,
    eCap: number,
  ): void {
    const buffer = this.metadataBuffer;
    this.timelineIds = new Uint32Array(buffer, layout.timelineIds, tCap);
    this.timelineTypeIds = new Uint32Array(
      buffer,
      layout.timelineTypeIds,
      tCap,
    );
    this.timelineNameStringIds = new Uint32Array(
      buffer,
      layout.timelineNameStringIds,
      tCap,
    );
    this.timelineParentIds = new Uint32Array(
      buffer,
      layout.timelineParentIds,
      tCap,
    );

    this.revisionIds = new Uint32Array(buffer, layout.revisionIds, rCap);
    this.revisionLogIds = new Uint32Array(buffer, layout.revisionLogIds, rCap);
    this.revisionPrincipalStringIds = new Uint32Array(
      buffer,
      layout.revisionPrincipalStringIds,
      rCap,
    );
    this.revisionVerbTypeIds = new Uint32Array(
      buffer,
      layout.revisionVerbTypeIds,
      rCap,
    );
    this.revisionStateTypeIds = new Uint32Array(
      buffer,
      layout.revisionStateTypeIds,
      rCap,
    );
    this.revisionBodyStructIds = new Uint32Array(
      buffer,
      layout.revisionBodyStructIds,
      rCap,
    );
    this.revisionChangedTimes = new BigUint64Array(
      buffer,
      layout.revisionChangedTimes,
      rCap,
    );

    this.eventIds = new Uint32Array(buffer, layout.eventIds, eCap);
    this.eventLogIds = new Uint32Array(buffer, layout.eventLogIds, eCap);
  }

  /**
   * Loads the raw timelines, revisions, and events into allocated metadata.
   */
  private load(
    timelines: Iterable<TimelineDTO>,
    timelineCount: number,
    revisions: Iterable<RevisionDTO>,
    revisionCount: number,
    events: Iterable<EventDTO>,
    eventCount: number,
  ): void {
    this.allocateMetadata(timelineCount, revisionCount, eventCount);

    this.timelineRevisionIds.length = 0;
    this.timelineEventIds.length = 0;
    this.timelinesList.length = 0;
    this.timelineChildrenIds.length = 0;
    this.revisionFieldAnnotations.length = revisionCount;
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

    // Load revisions
    let rIndex = 0;
    for (const r of revisions) {
      this.revisionIds[rIndex] = r.id;
      this.revisionLogIds[rIndex] = r.logId;
      this.revisionChangedTimes[rIndex] = r.changedTime;
      this.revisionPrincipalStringIds[rIndex] = r.principalStringId;
      this.revisionVerbTypeIds[rIndex] = r.verbTypeId;
      this.revisionStateTypeIds[rIndex] = r.stateTypeId;
      this.revisionBodyStructIds[rIndex] = r.resourceBodyStructId ?? 0;
      this.revisionFieldAnnotations[rIndex] = r.fieldAnnotations;

      this.revisionIdToIndex[r.id] = rIndex;
      rIndex++;
    }

    // Load events
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
    revisions.sort((r1, r2) => r1.logIndex - r2.logIndex);
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
   * Gets the interned struct ID of a revision body, or 0 if not stored as a struct.
   */
  public getRevisionBodyStructId(id: number): number {
    const index = this.getRevisionIndex(id);
    return this.revisionBodyStructIds[index] ?? 0;
  }

  /**
   * Gets the field annotations associated with this revision.
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionFieldAnnotations(
    id: number,
  ): readonly DomainFieldAnnotation[] {
    const raw = this.revisionFieldAnnotations[this.getRevisionIndex(id)];
    if (!raw) {
      return [];
    }
    return raw.map((fa: FieldAnnotationDTO) => {
      const fieldPath = this.internPool.getString(fa.fieldPathStringId);
      if (fa.mutatingWebhook) {
        return {
          fieldPath,
          mutatingWebhook: {
            configuration: this.internPool.getString(
              fa.mutatingWebhook.configurationStringId,
            ),
            webhook: this.internPool.getString(
              fa.mutatingWebhook.webhookStringId,
            ),
            round: fa.mutatingWebhook.round,
            index: fa.mutatingWebhook.index,
          },
        };
      }
      return { fieldPath };
    });
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
