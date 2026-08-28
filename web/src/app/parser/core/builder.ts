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

import { InspectionData } from 'src/app/store/domain/inspection-data';
import { LogStore, LogDTO } from 'src/app/store/domain/log-store';
import {
  InternPoolStore,
  StringEntryDTO,
} from 'src/app/store/domain/intern-pool-store';
import { IconAtlasDTO, StyleStore } from 'src/app/store/domain/style-store';
import {
  TimelineStore,
  RevisionDTO,
  TimelineDTO,
  EventDTO,
} from 'src/app/store/domain/timeline-store';
import {
  FieldPathSetDTO,
  StructDTO,
  StructStore,
} from 'src/app/store/domain/struct-store';
import {
  LogType,
  RevisionState,
  Severity,
  TimelineType,
  Verb,
} from 'src/app/store/domain/style';
import {
  InspectionHeader,
  InspectionQuery,
  MetadataStore,
} from 'src/app/store/domain/metadata-store';

/**
 * Core InspectionDataBuilder for compiling raw store inputs.
 * Directly populates domain stores incrementally during file ingestion.
 */
export class InspectionDataBuilder {
  private readonly internPool = InternPoolStore.create();
  private readonly styleStore = new StyleStore();
  private readonly logStore: LogStore;
  private readonly timelineStore: TimelineStore;
  private readonly structStore: StructStore;

  private readonly metadataStore: MetadataStore = {
    header: undefined,
    queries: [],
  };

  private iconAtlasPromise?: Promise<void>;

  /**
   * Initializes the InspectionDataBuilder with linked domain stores.
   */
  constructor() {
    this.logStore = LogStore.create(this.internPool, this.styleStore);
    this.timelineStore = TimelineStore.create(
      this.internPool,
      this.styleStore,
      this.logStore,
    );
    this.structStore = StructStore.create(this.internPool);
  }

  /**
   * Adds an interned string entry to the pool.
   */
  public addString(entry: StringEntryDTO): this {
    this.internPool.addString(entry);
    return this;
  }

  /**
   * Adds a single domain log entry.
   */
  public addLog(log: LogDTO): this {
    this.logStore.addLog(log);
    return this;
  }

  /**
   * Adds a single timeline definition.
   */
  public addTimeline(timeline: TimelineDTO): this {
    this.timelineStore.addTimeline(timeline);
    return this;
  }

  /**
   * Adds a single revision.
   */
  public addRevision(revision: RevisionDTO): this {
    this.timelineStore.addRevision(revision);
    return this;
  }

  /**
   * Adds a single event.
   */
  public addEvent(event: EventDTO): this {
    this.timelineStore.addEvent(event);
    return this;
  }

  /**
   * Adds a FieldPathSet to the struct store.
   */
  public addFieldPathSet(set: FieldPathSetDTO): this {
    this.structStore.addFieldPathSet(set);
    return this;
  }

  /**
   * Adds an InternedStruct to the struct store.
   */
  public addStruct(struct: StructDTO): this {
    this.structStore.addStruct(struct);
    return this;
  }

  /**
   * Registers styling metadata: severity rules.
   */
  public addSeverities(items: Iterable<Severity>): this {
    this.styleStore.addSeverities(items);
    return this;
  }

  /**
   * Registers styling metadata: logging types.
   */
  public addLogTypes(items: Iterable<LogType>): this {
    this.styleStore.addLogTypes(items);
    return this;
  }

  /**
   * Registers styling metadata: verb classifications.
   */
  public addVerbs(items: Iterable<Verb>): this {
    this.styleStore.addVerbs(items);
    return this;
  }

  /**
   * Registers styling metadata: revision tracking states.
   */
  public addRevisionStates(items: Iterable<RevisionState>): this {
    this.styleStore.addRevisionStates(items);
    return this;
  }

  /**
   * Registers styling metadata: classification types.
   */
  public addTimelineTypes(items: Iterable<TimelineType>): this {
    this.styleStore.addTimelineTypes(items);
    return this;
  }

  /**
   * Sets the icon atlas and tracks the asynchronous loading promise.
   */
  public setIconAtlas(dto: IconAtlasDTO): this {
    this.iconAtlasPromise = this.styleStore.setIconAtlas(dto);
    this.iconAtlasPromise.catch(() => {}); // Prevents unhandled rejection.
    return this;
  }

  /**
   * Sets the primary inspection metadata header.
   */
  public setMetadataHeader(header: InspectionHeader): this {
    this.metadataStore.header = header;
    return this;
  }

  /**
   * Adds saved inspection queries to the collection.
   */
  public addMetadataQueries(queries: Iterable<InspectionQuery>): this {
    for (const q of queries) {
      this.metadataStore.queries.push(q);
    }
    return this;
  }

  /**
   * Compacts all domain stores and returns the completed InspectionData.
   */
  public async build(): Promise<InspectionData> {
    this.internPool.shrinkToFit();
    this.logStore.shrinkToFit();
    this.timelineStore.shrinkToFit();
    this.structStore.shrinkToFit();

    if (this.iconAtlasPromise) {
      await this.iconAtlasPromise;
    }

    return {
      internPool: this.internPool,
      styleStore: this.styleStore,
      logStore: this.logStore,
      timelineStore: this.timelineStore,
      structStore: this.structStore,
      metadata: this.metadataStore,
    };
  }
}
