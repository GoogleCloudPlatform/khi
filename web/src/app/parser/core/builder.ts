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
 * Collects components in a version-decoupled form.
 */
export class InspectionDataBuilder {
  private readonly styleStore = new StyleStore();
  private readonly metadataStore: MetadataStore = {
    header: undefined,
    queries: [],
  };

  private readonly rawStrings: StringEntryDTO[] = [];
  private readonly rawLogs: LogDTO[] = [];
  private readonly rawTimelines: TimelineDTO[] = [];
  private readonly rawRevisions: RevisionDTO[] = [];
  private readonly rawEvents: EventDTO[] = [];

  private iconAtlasPromise?: Promise<void>;

  /**
   * Adds interned strings to the pool.
   */
  public addStrings(strings: Iterable<StringEntryDTO>): this {
    for (const s of strings) {
      this.rawStrings.push(s);
    }
    return this;
  }

  /**
   * Adds domain raw logs.
   */
  public addLogs(logs: Iterable<LogDTO>): this {
    for (const log of logs) {
      this.rawLogs.push(log);
    }
    return this;
  }

  /**
   * Adds domain raw timelines.
   */
  public addTimelines(timelines: Iterable<TimelineDTO>): this {
    for (const timeline of timelines) {
      this.rawTimelines.push(timeline);
    }
    return this;
  }

  /**
   * Adds domain raw revisions.
   */
  public addRevisions(revisions: Iterable<RevisionDTO>): this {
    for (const revision of revisions) {
      this.rawRevisions.push(revision);
    }
    return this;
  }

  /**
   * Adds domain raw events.
   */
  public addEvents(events: Iterable<EventDTO>): this {
    for (const event of events) {
      this.rawEvents.push(event);
    }
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
    this.iconAtlasPromise.catch(() => {}); // Prevents the unhandled rejection. Error will be thrown in the build method to actually await the promise.
    return this;
  }

  /**
   * Sets the primary inspection metadata header.
   */
  public setMetadataHeader(header: InspectionHeader): void {
    this.metadataStore.header = header;
  }

  /**
   * Adds saved inspection queries to the collection.
   */
  public addMetadataQueries(queries: Iterable<InspectionQuery>): void {
    for (const q of queries) {
      this.metadataStore.queries.push(q);
    }
  }

  /**
   * Retrieves the StyleStore instance managed by this builder.
   */
  public getStyleStore(): StyleStore {
    return this.styleStore;
  }

  /**
   * Instantiates data store contexts returning root inspection model.
   */
  public async build(): Promise<InspectionData> {
    const internPool = InternPoolStore.initialize(this.rawStrings);
    const logStore = LogStore.initialize(
      internPool,
      this.styleStore,
      this.rawLogs,
      this.rawLogs.length,
    );
    const timelineStore = TimelineStore.initialize(
      internPool,
      this.styleStore,
      logStore,
      this.rawTimelines,
      this.rawTimelines.length,
      this.rawRevisions,
      this.rawRevisions.length,
      this.rawEvents,
      this.rawEvents.length,
    );

    if (this.iconAtlasPromise) {
      await this.iconAtlasPromise;
    }

    return {
      internPool,
      styleStore: this.styleStore,
      logStore,
      timelineStore,
      metadata: this.metadataStore,
    };
  }
}
