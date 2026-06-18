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
import { InternedStructDecoder } from 'src/app/store/domain/struct-decoder';
import { InternedStructSchema } from 'src/app/generated/khifile/shared_pb';
import {
  RevisionState,
  Severity,
  TimelineType,
  Verb,
  StyleProvider,
} from 'src/app/store/domain/style';
import { LogStore } from 'src/app/store/domain/log-store';
import {
  ReadonlyDomainElement,
  allocateBuffer,
  isSharedBuffer,
} from 'src/app/store/domain/types';
import { fromBinary } from '@bufbuild/protobuf';

/**
 * Align the offset to the specified byte alignment.
 */
function align(offset: number, alignment: number): number {
  return Math.ceil(offset / alignment) * alignment;
}

/**
 * Represents the shared memory structure of the timeline store.
 * This is used to pass data to the worker threads.
 * To prevent the main thread from OOM killing, the main contents are shared via SharedArrayBuffer.
 */
export interface TimelineStoreSharedData {
  readonly metadataSab: SharedArrayBuffer | ArrayBuffer;
  readonly bodyBufferSabs: readonly (SharedArrayBuffer | ArrayBuffer)[];
  readonly timelineCount: number;
  readonly revisionCount: number;
  readonly eventCount: number;
  readonly timelineRevisionIds: Uint32Array[];
  readonly timelineEventIds: Uint32Array[];
  readonly timelineIdToIndex: { readonly [tid: number]: number };
  readonly revisionIdToIndex: { readonly [rid: number]: number };
  readonly eventIdToIndex: { readonly [eid: number]: number };
}

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
  readonly body?: Uint8Array;
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
  private readonly readOnly: boolean;

  private metadataSab!: SharedArrayBuffer | ArrayBuffer;

  // Timeline views
  private timelineIds!: Uint32Array;
  private timelineTypeIds!: Uint32Array;
  private timelineNameStringIds!: Uint32Array;
  private timelineParentIds!: Uint32Array;
  private timelineSeverities!: Uint8Array;

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

  // Packed revision bodies
  private revisionBodyBufferIndices!: Uint16Array;
  private revisionBodyOffsets!: Uint32Array;
  private revisionBodyLengths!: Uint32Array;

  private readonly revisionBodyBufferSabs: (SharedArrayBuffer | ArrayBuffer)[] =
    [];
  private readonly revisionBodyBuffers: Uint8Array[] = [];

  private currentBufferIndex = -1;
  private currentOffset = 0;

  private readonly revisionDecodedBodyCache: WeakRef<
    ReadonlyDomainElement<Record<string, unknown>>
  >[] = [];
  private revisionIdToIndex: { [rid: number]: number } = {};

  // Event views
  private eventIds!: Uint32Array;
  private eventLogIds!: Uint32Array;
  private eventIdToIndex: { [eid: number]: number } = {};

  private readonly decoder: InternedStructDecoder;

  private constructor(
    private readonly internPool: InternPoolStore,
    public readonly styleStore: StyleProvider,
    public readonly logStore: LogStore,
    private readonly maxBufferSize: number,
    readOnly: boolean,
    initialData:
      | { timelineCount: number; revisionCount: number; eventCount: number }
      | TimelineStoreSharedData,
  ) {
    this.readOnly = readOnly;
    this.decoder = new InternedStructDecoder(this.internPool);

    if ('metadataSab' in initialData) {
      const sharedData = initialData;
      this.metadataSab = sharedData.metadataSab;
      this.revisionBodyBufferSabs = Array.from(sharedData.bodyBufferSabs);
      this.revisionBodyBuffers = this.revisionBodyBufferSabs.map(
        (sab) => new Uint8Array(sab),
      );

      this.timelineRevisionIds = sharedData.timelineRevisionIds;
      this.timelineEventIds = sharedData.timelineEventIds;

      this.timelineIdToIndex = sharedData.timelineIdToIndex;
      this.revisionIdToIndex = sharedData.revisionIdToIndex;
      this.eventIdToIndex = sharedData.eventIdToIndex;

      this.mapMetadataViews(
        sharedData.timelineCount,
        sharedData.revisionCount,
        sharedData.eventCount,
      );

      this.timelineChildrenIds = [];
      for (let i = 0; i < sharedData.timelineCount; i++) {
        this.timelineChildrenIds[i] = [];
      }

      for (let i = 0; i < sharedData.timelineCount; i++) {
        const parentId = this.timelineParentIds[i];
        if (parentId !== 0) {
          const parentIndex = this.timelineIdToIndex[parentId];
          if (parentIndex !== undefined) {
            this.timelineChildrenIds[parentIndex].push(this.timelineIds[i]);
          }
        }
      }

      for (let i = 0; i < sharedData.timelineCount; i++) {
        this.timelinesList.push(new Timeline(this.timelineIds[i], this));
      }
    } else {
      const counts = initialData;
      this.allocateMetadata(
        counts.timelineCount,
        counts.revisionCount,
        counts.eventCount,
      );
    }
  }

  /**
   * Creates a new writable TimelineStore instance.
   */
  public static create(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    logStore: LogStore,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): TimelineStore {
    return new TimelineStore(
      internPool,
      styleStore,
      logStore,
      maxBufferSize,
      false,
      { timelineCount: 1024, revisionCount: 1024, eventCount: 1024 },
    );
  }

  /**
   * Reconstructs a read-only TimelineStore instance from shared memory data.
   */
  public static fromSharedData(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    logStore: LogStore,
    sharedData: TimelineStoreSharedData,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): TimelineStore {
    return new TimelineStore(
      internPool,
      styleStore,
      logStore,
      maxBufferSize,
      true,
      sharedData,
    );
  }

  private allocateMetadata(tCap: number, rCap: number, eCap: number): void {
    const layout = this.calculateOffsets(tCap, rCap, eCap);
    this.metadataSab = allocateBuffer(layout.totalBytes);
    this.applyViews(layout, tCap, rCap, eCap);
  }

  private mapMetadataViews(tCap: number, rCap: number, eCap: number): void {
    const layout = this.calculateOffsets(tCap, rCap, eCap);
    this.applyViews(layout, tCap, rCap, eCap);
  }

  private calculateOffsets(tCap: number, rCap: number, eCap: number) {
    let offset = 0;

    const timelineIds = offset;
    offset += tCap * 4;

    const timelineTypeIds = offset;
    offset += tCap * 4;

    const timelineNameStringIds = offset;
    offset += tCap * 4;

    const timelineParentIds = offset;
    offset += tCap * 4;

    const timelineSeverities = offset;
    offset += tCap * 1;

    const revisionIds = align(offset, 4);
    offset = revisionIds + rCap * 4;

    const revisionLogIds = offset;
    offset += rCap * 4;

    const revisionChangedTimes = align(offset, 8);
    offset = revisionChangedTimes + rCap * 8;

    const revisionPrincipalStringIds = offset;
    offset += rCap * 4;

    const revisionVerbTypeIds = offset;
    offset += rCap * 4;

    const revisionStateTypeIds = offset;
    offset += rCap * 4;

    const revisionBodyBufferIndices = align(offset, 2);
    offset = revisionBodyBufferIndices + rCap * 2;

    const revisionBodyOffsets = align(offset, 4);
    offset = revisionBodyOffsets + rCap * 4;

    const revisionBodyLengths = align(offset, 4);
    offset = revisionBodyLengths + rCap * 4;

    const eventIds = align(offset, 4);
    offset = eventIds + eCap * 4;

    const eventLogIds = offset;
    offset += eCap * 4;

    return {
      timelineIds,
      timelineTypeIds,
      timelineNameStringIds,
      timelineParentIds,
      timelineSeverities,
      revisionIds,
      revisionLogIds,
      revisionChangedTimes,
      revisionPrincipalStringIds,
      revisionVerbTypeIds,
      revisionStateTypeIds,
      revisionBodyBufferIndices,
      revisionBodyOffsets,
      revisionBodyLengths,
      eventIds,
      eventLogIds,
      totalBytes: offset,
    };
  }

  private applyViews(
    layout: ReturnType<typeof TimelineStore.prototype.calculateOffsets>,
    tCap: number,
    rCap: number,
    eCap: number,
  ): void {
    const sab = this.metadataSab;
    this.timelineIds = new Uint32Array(sab, layout.timelineIds, tCap);
    this.timelineTypeIds = new Uint32Array(sab, layout.timelineTypeIds, tCap);
    this.timelineNameStringIds = new Uint32Array(
      sab,
      layout.timelineNameStringIds,
      tCap,
    );
    this.timelineParentIds = new Uint32Array(
      sab,
      layout.timelineParentIds,
      tCap,
    );
    this.timelineSeverities = new Uint8Array(
      sab,
      layout.timelineSeverities,
      tCap,
    );

    this.revisionIds = new Uint32Array(sab, layout.revisionIds, rCap);
    this.revisionLogIds = new Uint32Array(sab, layout.revisionLogIds, rCap);
    this.revisionChangedTimes = new BigUint64Array(
      sab,
      layout.revisionChangedTimes,
      rCap,
    );
    this.revisionPrincipalStringIds = new Uint32Array(
      sab,
      layout.revisionPrincipalStringIds,
      rCap,
    );
    this.revisionVerbTypeIds = new Uint32Array(
      sab,
      layout.revisionVerbTypeIds,
      rCap,
    );
    this.revisionStateTypeIds = new Uint32Array(
      sab,
      layout.revisionStateTypeIds,
      rCap,
    );
    this.revisionBodyBufferIndices = new Uint16Array(
      sab,
      layout.revisionBodyBufferIndices,
      rCap,
    );
    this.revisionBodyOffsets = new Uint32Array(
      sab,
      layout.revisionBodyOffsets,
      rCap,
    );
    this.revisionBodyLengths = new Uint32Array(
      sab,
      layout.revisionBodyLengths,
      rCap,
    );

    this.eventIds = new Uint32Array(sab, layout.eventIds, eCap);
    this.eventLogIds = new Uint32Array(sab, layout.eventLogIds, eCap);
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
    if (this.readOnly) {
      throw new Error('Cannot write to a shared read-only TimelineStore');
    }

    this.allocateMetadata(timelineCount, revisionCount, eventCount);

    this.timelineRevisionIds.length = 0;
    this.timelineEventIds.length = 0;
    this.revisionBodyBufferSabs.length = 0;
    this.revisionBodyBuffers.length = 0;
    this.timelinesList.length = 0;
    this.timelineChildrenIds.length = 0;
    this.timelineIdToIndex = {};
    this.revisionIdToIndex = {};
    this.eventIdToIndex = {};

    this.currentBufferIndex = -1;
    this.currentOffset = 0;

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

    this.revisionDecodedBodyCache.length = revisionCount;

    // Load revisions
    let rIndex = 0;
    for (const r of revisions) {
      this.revisionIds[rIndex] = r.id;
      this.revisionLogIds[rIndex] = r.logId;
      this.revisionChangedTimes[rIndex] = r.changedTime;
      this.revisionPrincipalStringIds[rIndex] = r.principalStringId;
      this.revisionVerbTypeIds[rIndex] = r.verbTypeId;
      this.revisionStateTypeIds[rIndex] = r.stateTypeId;

      if (r.body !== undefined && r.body.length > 0) {
        this.addRevisionBody(rIndex, r.body);
      } else {
        this.revisionBodyBufferIndices[rIndex] = 0;
        this.revisionBodyOffsets[rIndex] = 0;
        this.revisionBodyLengths[rIndex] = 0;
      }
      this.revisionIdToIndex[r.id] = rIndex;
      rIndex++;
    }

    // load events
    let eIndex = 0;
    for (const e of events) {
      this.eventIds[eIndex] = e.id;
      this.eventLogIds[eIndex] = e.logId;
      this.eventIdToIndex[e.id] = eIndex;
      eIndex++;
    }

    // Build severity index for timelines
    for (let i = 0; i < timelineCount; i++) {
      let mask = 0;
      const revIds = this.timelineRevisionIds[i];
      for (let j = 0; j < revIds.length; j++) {
        const rIndex = this.revisionIdToIndex[revIds[j]];
        if (rIndex !== undefined) {
          const logId = this.revisionLogIds[rIndex];
          const severityId = this.logStore._getSeverity(logId).id;
          if (severityId >= 0 && severityId < 8) {
            mask |= 1 << severityId;
          }
        }
      }
      const eventIds = this.timelineEventIds[i];
      for (let j = 0; j < eventIds.length; j++) {
        const eIndex = this.eventIdToIndex[eventIds[j]];
        if (eIndex !== undefined) {
          const logId = this.eventLogIds[eIndex];
          const severityId = this.logStore._getSeverity(logId).id;
          if (severityId >= 0 && severityId < 8) {
            mask |= 1 << severityId;
          }
        }
      }
      this.timelineSeverities[i] = mask;
    }
  }

  private addRevisionBody(index: number, bodyBytes: Uint8Array): void {
    const length = bodyBytes.length;

    if (length > this.maxBufferSize) {
      const sab = allocateBuffer(length);
      const buf = new Uint8Array(sab);
      buf.set(bodyBytes);
      this.revisionBodyBufferSabs.push(sab);
      this.revisionBodyBuffers.push(buf);

      const newBufIdx = this.revisionBodyBuffers.length - 1;
      this.revisionBodyBufferIndices[index] = newBufIdx + 1;
      this.revisionBodyOffsets[index] = 0;
      this.revisionBodyLengths[index] = length;
      return;
    }

    if (
      this.currentBufferIndex === -1 ||
      this.currentOffset + length > this.maxBufferSize
    ) {
      const sab = allocateBuffer(this.maxBufferSize);
      const buf = new Uint8Array(sab);
      this.revisionBodyBufferSabs.push(sab);
      this.revisionBodyBuffers.push(buf);
      this.currentBufferIndex = this.revisionBodyBuffers.length - 1;
      this.currentOffset = 0;
    }

    const currentBuf = this.revisionBodyBuffers[this.currentBufferIndex];
    currentBuf.set(bodyBytes, this.currentOffset);

    this.revisionBodyBufferIndices[index] = this.currentBufferIndex + 1;
    this.revisionBodyOffsets[index] = this.currentOffset;
    this.revisionBodyLengths[index] = length;

    this.currentOffset += length;
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

  /**
   * Checks if the timeline with the specified ID has any logs with any of the specified severities.
   *
   * @note Intended solely for internal retrieval inside the {@link Timeline} domain adapter.
   */
  public _hasSeverities(
    id: number,
    severities: readonly ReadonlyDomainElement<Severity>[],
  ): boolean {
    const index = this.getTimelineIndex(id);
    let severityMask = 0;
    for (const s of severities) {
      if (s.id >= 0 && s.id < 8) {
        severityMask |= 1 << s.id;
      }
    }
    return (this.timelineSeverities[index] & severityMask) !== 0;
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
    const index = this.getRevisionIndex(id);
    const cached = this.revisionDecodedBodyCache[index]?.deref();
    if (cached) {
      return cached;
    }

    const bufIdx = this.revisionBodyBufferIndices[index];
    if (bufIdx === 0) return null;

    const offset = this.revisionBodyOffsets[index];
    const length = this.revisionBodyLengths[index];

    const buffer = this.revisionBodyBuffers[bufIdx - 1];
    const bytes = buffer.subarray(offset, offset + length);
    let decodeTarget = bytes;
    if (isSharedBuffer(bytes.buffer)) {
      const nonSharedBytes = new Uint8Array(length);
      nonSharedBytes.set(bytes);
      decodeTarget = nonSharedBytes;
    }

    const struct = fromBinary(InternedStructSchema, decodeTarget);
    const decoded = this.decoder.decode(struct);
    this.revisionDecodedBodyCache[index] = new WeakRef(decoded);
    return decoded;
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

  /**
   * Returns the shared memory representation of this TimelineStore.
   */
  public getSharedData(): TimelineStoreSharedData {
    return {
      metadataSab: this.metadataSab,
      bodyBufferSabs: this.revisionBodyBufferSabs,
      timelineCount: this.timelineIds.length,
      revisionCount: this.revisionIds.length,
      eventCount: this.eventIds.length,
      timelineRevisionIds: this.timelineRevisionIds,
      timelineEventIds: this.timelineEventIds,
      timelineIdToIndex: this.timelineIdToIndex,
      revisionIdToIndex: this.revisionIdToIndex,
      eventIdToIndex: this.eventIdToIndex,
    };
  }
}
