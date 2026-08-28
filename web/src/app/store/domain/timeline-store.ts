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
import {
  BufferLayoutBuilder,
  nextCapacity,
} from 'src/app/store/domain/buffer-util';
import { LogStore } from 'src/app/store/domain/log-store';
import {
  DomainFieldAnnotation,
  ReadonlyDomainElement,
  allocateBuffer,
} from 'src/app/store/domain/types';

const EMPTY_UINT32_ARRAY = new Uint32Array(0);

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

export interface MutatingWebhookAnnotationDTO {
  readonly configurationStringId: number;
  readonly webhookStringId: number;
  readonly round: number;
  readonly index: number;
}

export interface FieldAnnotationDTO {
  readonly fieldPathStringId: number;
  readonly mutatingWebhook?: MutatingWebhookAnnotationDTO;
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
  readonly timelineIdsOffset: number;
  readonly timelineTypeIdsOffset: number;
  readonly timelineNameStringIdsOffset: number;
  readonly timelineParentIdsOffset: number;
  readonly revisionLogIdsOffset: number;
  readonly revisionPrincipalStringIdsOffset: number;
  readonly revisionVerbTypeIdsOffset: number;
  readonly revisionStateTypeIdsOffset: number;
  readonly revisionBodyStructIdsOffset: number;
  readonly revisionChangedTimesOffset: number;
  readonly eventLogIdsOffset: number;
  readonly totalBytes: number;
}

/**
 * Store for managing and retrieving timelines, revisions, and events efficiently.
 */
export class TimelineStore {
  private metadataBuffer!: ArrayBuffer;
  private timelineCapacity = 0;
  private revisionCapacity = 0;
  private eventCapacity = 0;

  private timelineCount = 0;
  private _revisionCount = 0;
  private _eventCount = 0;
  private relationshipsDirty = true;

  // Timeline views
  private timelineIds!: Uint32Array;
  private timelineTypeIds!: Uint32Array;
  private timelineNameStringIds!: Uint32Array;
  private timelineParentIds!: Uint32Array;

  private readonly timelineRevisionIds: Uint32Array[] = [];
  private readonly timelineEventIds: Uint32Array[] = [];
  private readonly timelineChildrenIds: number[][] = [];
  private readonly timelinesList: ReadonlyDomainElement<Timeline>[] = [];
  private timelineIdToIndex: (number | undefined)[] = [];

  // Revision views
  private revisionLogIds!: Uint32Array;
  private revisionChangedTimes!: BigUint64Array;
  private revisionPrincipalStringIds!: Uint32Array;
  private revisionVerbTypeIds!: Uint32Array;
  private revisionStateTypeIds!: Uint32Array;
  private revisionBodyStructIds!: Uint32Array;
  private readonly revisionFieldAnnotations: (
    readonly FieldAnnotationDTO[] | undefined
  )[] = [];
  private revisionIdToIndex: (number | undefined)[] = [];

  // Event views
  private eventLogIds!: Uint32Array;
  private eventIdToIndex: (number | undefined)[] = [];

  private constructor(
    private readonly internPool: InternPoolStore,
    public readonly styleStore: StyleProvider,
    public readonly logStore: LogStore,
  ) {}

  /**
   * Creates an empty TimelineStore instance with initial capacities.
   *
   * @param internPool Intern pool store for string interning.
   * @param styleStore Style provider for styling information.
   * @param logStore Log store providing log references.
   * @param initialTimelineCapacity Initial capacity for timelines.
   * @param initialRevisionCapacity Initial capacity for revisions.
   * @param initialEventCapacity Initial capacity for events.
   * @returns A new empty TimelineStore instance.
   */
  public static create(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    logStore: LogStore,
    initialTimelineCapacity = 1024,
    initialRevisionCapacity = 1024,
    initialEventCapacity = 1024,
  ): TimelineStore {
    const store = new TimelineStore(internPool, styleStore, logStore);
    store.ensureCapacity(
      Math.max(initialTimelineCapacity, 1),
      Math.max(initialRevisionCapacity, 1),
      Math.max(initialEventCapacity, 1),
    );
    return store;
  }

  /**
   * Gets the total number of timelines.
   */
  public get count(): number {
    return this.timelineCount;
  }

  /**
   * Gets the total number of revisions.
   */
  public get revisionCount(): number {
    return this._revisionCount;
  }

  /**
   * Gets the total number of events.
   */
  public get eventCount(): number {
    return this._eventCount;
  }

  /**
   * Ensures timeline metadata has at least minCapacity slots.
   */
  private ensureTimelineCapacity(minCapacity: number): void {
    this.ensureCapacity(minCapacity, this.revisionCapacity, this.eventCapacity);
  }

  /**
   * Ensures revision metadata has at least minCapacity slots.
   */
  private ensureRevisionCapacity(minCapacity: number): void {
    this.ensureCapacity(this.timelineCapacity, minCapacity, this.eventCapacity);
  }

  /**
   * Ensures event metadata has at least minCapacity slots.
   */
  private ensureEventCapacity(minCapacity: number): void {
    this.ensureCapacity(
      this.timelineCapacity,
      this.revisionCapacity,
      minCapacity,
    );
  }

  private calculateOffsets(
    timelineCapacity: number,
    revisionCapacity: number,
    eventCapacity: number,
  ): TimelineStoreLayout {
    const builder = new BufferLayoutBuilder();
    return {
      timelineIdsOffset: builder.addField(timelineCapacity, 4),
      timelineTypeIdsOffset: builder.addField(timelineCapacity, 4),
      timelineNameStringIdsOffset: builder.addField(timelineCapacity, 4),
      timelineParentIdsOffset: builder.addField(timelineCapacity, 4),
      revisionLogIdsOffset: builder.addField(revisionCapacity, 4, 4),
      revisionPrincipalStringIdsOffset: builder.addField(revisionCapacity, 4),
      revisionVerbTypeIdsOffset: builder.addField(revisionCapacity, 4),
      revisionStateTypeIdsOffset: builder.addField(revisionCapacity, 4),
      revisionBodyStructIdsOffset: builder.addField(revisionCapacity, 4),
      revisionChangedTimesOffset: builder.addField(revisionCapacity, 8, 8),
      eventLogIdsOffset: builder.addField(eventCapacity, 4, 4),
      totalBytes: builder.totalBytes,
    };
  }

  private applyViews(
    layout: TimelineStoreLayout,
    timelineCapacity: number,
    revisionCapacity: number,
    eventCapacity: number,
  ): void {
    const buffer = this.metadataBuffer;
    this.timelineIds = new Uint32Array(
      buffer,
      layout.timelineIdsOffset,
      timelineCapacity,
    );
    this.timelineTypeIds = new Uint32Array(
      buffer,
      layout.timelineTypeIdsOffset,
      timelineCapacity,
    );
    this.timelineNameStringIds = new Uint32Array(
      buffer,
      layout.timelineNameStringIdsOffset,
      timelineCapacity,
    );
    this.timelineParentIds = new Uint32Array(
      buffer,
      layout.timelineParentIdsOffset,
      timelineCapacity,
    );

    this.revisionLogIds = new Uint32Array(
      buffer,
      layout.revisionLogIdsOffset,
      revisionCapacity,
    );
    this.revisionPrincipalStringIds = new Uint32Array(
      buffer,
      layout.revisionPrincipalStringIdsOffset,
      revisionCapacity,
    );
    this.revisionVerbTypeIds = new Uint32Array(
      buffer,
      layout.revisionVerbTypeIdsOffset,
      revisionCapacity,
    );
    this.revisionStateTypeIds = new Uint32Array(
      buffer,
      layout.revisionStateTypeIdsOffset,
      revisionCapacity,
    );
    this.revisionBodyStructIds = new Uint32Array(
      buffer,
      layout.revisionBodyStructIdsOffset,
      revisionCapacity,
    );
    this.revisionChangedTimes = new BigUint64Array(
      buffer,
      layout.revisionChangedTimesOffset,
      revisionCapacity,
    );

    this.eventLogIds = new Uint32Array(
      buffer,
      layout.eventLogIdsOffset,
      eventCapacity,
    );
  }

  private reallocate(
    newTimelineCapacity: number,
    newRevisionCapacity: number,
    newEventCapacity: number,
  ): void {
    const layout = this.calculateOffsets(
      newTimelineCapacity,
      newRevisionCapacity,
      newEventCapacity,
    );
    const newBuffer = allocateBuffer(layout.totalBytes);

    const prevTimelineIds = this.timelineIds;
    const prevTimelineTypeIds = this.timelineTypeIds;
    const prevTimelineNameStringIds = this.timelineNameStringIds;
    const prevTimelineParentIds = this.timelineParentIds;

    const prevRevisionLogIds = this.revisionLogIds;
    const prevRevisionPrincipalStringIds = this.revisionPrincipalStringIds;
    const prevRevisionVerbTypeIds = this.revisionVerbTypeIds;
    const prevRevisionStateTypeIds = this.revisionStateTypeIds;
    const prevRevisionBodyStructIds = this.revisionBodyStructIds;
    const prevRevisionChangedTimes = this.revisionChangedTimes;

    const prevEventLogIds = this.eventLogIds;

    this.metadataBuffer = newBuffer;
    this.timelineCapacity = newTimelineCapacity;
    this.revisionCapacity = newRevisionCapacity;
    this.eventCapacity = newEventCapacity;
    this.applyViews(
      layout,
      newTimelineCapacity,
      newRevisionCapacity,
      newEventCapacity,
    );

    if (this.timelineCount > 0 && prevTimelineIds) {
      this.timelineIds.set(prevTimelineIds.subarray(0, this.timelineCount));
      this.timelineTypeIds.set(
        prevTimelineTypeIds.subarray(0, this.timelineCount),
      );
      this.timelineNameStringIds.set(
        prevTimelineNameStringIds.subarray(0, this.timelineCount),
      );
      this.timelineParentIds.set(
        prevTimelineParentIds.subarray(0, this.timelineCount),
      );
    }

    if (this._revisionCount > 0 && prevRevisionLogIds) {
      this.revisionLogIds.set(
        prevRevisionLogIds.subarray(0, this._revisionCount),
      );
      this.revisionPrincipalStringIds.set(
        prevRevisionPrincipalStringIds.subarray(0, this._revisionCount),
      );
      this.revisionVerbTypeIds.set(
        prevRevisionVerbTypeIds.subarray(0, this._revisionCount),
      );
      this.revisionStateTypeIds.set(
        prevRevisionStateTypeIds.subarray(0, this._revisionCount),
      );
      this.revisionBodyStructIds.set(
        prevRevisionBodyStructIds.subarray(0, this._revisionCount),
      );
      this.revisionChangedTimes.set(
        prevRevisionChangedTimes.subarray(0, this._revisionCount),
      );
    }

    if (this._eventCount > 0 && prevEventLogIds) {
      this.eventLogIds.set(prevEventLogIds.subarray(0, this._eventCount));
    }
  }

  /**
   * Ensures metadata buffer has at least the specified capacities for all views.
   */
  private ensureCapacity(
    minTimelineCapacity: number,
    minRevisionCapacity: number,
    minEventCapacity: number,
  ): void {
    if (
      minTimelineCapacity <= this.timelineCapacity &&
      minRevisionCapacity <= this.revisionCapacity &&
      minEventCapacity <= this.eventCapacity
    ) {
      return;
    }

    const newTCap =
      minTimelineCapacity > this.timelineCapacity
        ? nextCapacity(this.timelineCapacity, minTimelineCapacity)
        : this.timelineCapacity;
    const newRCap =
      minRevisionCapacity > this.revisionCapacity
        ? nextCapacity(this.revisionCapacity, minRevisionCapacity)
        : this.revisionCapacity;
    const newECap =
      minEventCapacity > this.eventCapacity
        ? nextCapacity(this.eventCapacity, minEventCapacity)
        : this.eventCapacity;

    this.reallocate(newTCap, newRCap, newECap);
  }

  /**
   * Shrinks the metadataBuffer to fit exact counts of timelines, revisions, and events.
   */
  public shrinkToFit(): void {
    this.rebuildRelationships();
    if (
      this.timelineCount === this.timelineCapacity &&
      this._revisionCount === this.revisionCapacity &&
      this._eventCount === this.eventCapacity
    ) {
      return;
    }

    this.reallocate(this.timelineCount, this._revisionCount, this._eventCount);
  }

  /**
   * Appends a single timeline to the store.
   *
   * @param timeline The raw TimelineDTO to add.
   */
  public addTimeline(timeline: TimelineDTO): void {
    this.ensureTimelineCapacity(this.timelineCount + 1);
    const tIndex = this.timelineCount;
    this.timelineIds[tIndex] = timeline.id;
    this.timelineTypeIds[tIndex] = timeline.timelineTypeId;
    this.timelineNameStringIds[tIndex] = timeline.nameStringId;
    this.timelineParentIds[tIndex] = timeline.parentTimelineId;

    this.timelineRevisionIds[tIndex] =
      timeline.revisionIds.length > 0
        ? new Uint32Array(timeline.revisionIds)
        : EMPTY_UINT32_ARRAY;
    this.timelineEventIds[tIndex] =
      timeline.eventIds.length > 0
        ? new Uint32Array(timeline.eventIds)
        : EMPTY_UINT32_ARRAY;
    this.timelineChildrenIds[tIndex] = [];

    this.timelineIdToIndex[timeline.id] = tIndex;
    this.timelinesList.push(new Timeline(timeline.id, this));
    this.timelineCount++;
    this.relationshipsDirty = true;
  }

  /**
   * Appends multiple timelines to the store.
   *
   * @param timelines An iterable of raw TimelineDTOs to add.
   */
  public addTimelines(timelines: Iterable<TimelineDTO>): void {
    if (Array.isArray(timelines)) {
      this.ensureTimelineCapacity(this.timelineCount + timelines.length);
    }
    for (const timeline of timelines) {
      this.addTimeline(timeline);
    }
  }

  /**
   * Appends a single revision to the store.
   *
   * @param revision The raw RevisionDTO to add.
   */
  public addRevision(revision: RevisionDTO): void {
    this.ensureRevisionCapacity(this._revisionCount + 1);
    const rIndex = this._revisionCount;
    this.revisionLogIds[rIndex] = revision.logId;
    this.revisionChangedTimes[rIndex] = revision.changedTime;
    this.revisionPrincipalStringIds[rIndex] = revision.principalStringId;
    this.revisionVerbTypeIds[rIndex] = revision.verbTypeId;
    this.revisionStateTypeIds[rIndex] = revision.stateTypeId;
    this.revisionBodyStructIds[rIndex] = revision.resourceBodyStructId ?? 0;
    this.revisionFieldAnnotations[rIndex] = revision.fieldAnnotations;

    this.revisionIdToIndex[revision.id] = rIndex;
    this._revisionCount++;
  }

  /**
   * Appends multiple revisions to the store.
   *
   * @param revisions An iterable of raw RevisionDTOs to add.
   */
  public addRevisions(revisions: Iterable<RevisionDTO>): void {
    if (Array.isArray(revisions)) {
      this.ensureRevisionCapacity(this._revisionCount + revisions.length);
    }
    for (const revision of revisions) {
      this.addRevision(revision);
    }
  }

  /**
   * Appends a single event to the store.
   *
   * @param event The raw EventDTO to add.
   */
  public addEvent(event: EventDTO): void {
    this.ensureEventCapacity(this._eventCount + 1);
    const eIndex = this._eventCount;
    this.eventLogIds[eIndex] = event.logId;

    this.eventIdToIndex[event.id] = eIndex;
    this._eventCount++;
  }

  /**
   * Appends multiple events to the store.
   *
   * @param events An iterable of raw EventDTOs to add.
   */
  public addEvents(events: Iterable<EventDTO>): void {
    if (Array.isArray(events)) {
      this.ensureEventCapacity(this._eventCount + events.length);
    }
    for (const event of events) {
      this.addEvent(event);
    }
  }

  /**
   * Rebuilds child timeline relationships based on parentTimelineId.
   */
  private rebuildRelationships(): void {
    if (!this.relationshipsDirty) {
      return;
    }
    for (let i = 0; i < this.timelineCount; i++) {
      this.timelineChildrenIds[i] = [];
    }
    for (let i = 0; i < this.timelineCount; i++) {
      const parentId = this.timelineParentIds[i];
      if (parentId !== 0) {
        const pIndex = this.timelineIdToIndex[parentId];
        if (pIndex !== undefined) {
          this.timelineChildrenIds[pIndex].push(this.timelineIds[i]);
        }
      }
    }
    this.relationshipsDirty = false;
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
    if (this.relationshipsDirty) {
      this.rebuildRelationships();
    }
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
   * @note Intended solely for internal retrieval inside the {@link Revision} domain adapter.
   */
  public _getRevisionBodyStructId(id: number): number {
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
