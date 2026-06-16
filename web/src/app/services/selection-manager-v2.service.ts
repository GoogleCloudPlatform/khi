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
import { Injectable, inject, signal, computed } from '@angular/core';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { Log } from 'src/app/store/domain/log';
import { Timeline, Revision, Event } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * SelectionManagerV2 provides selected/highlighted list of logs, timelines, revisions or events from the received user interaction.
 * Signal-based modern version of SelectionManagerService.
 */
@Injectable({ providedIn: 'root' })
export class SelectionManagerV2 {
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);

  // Writable signals representing internal state.
  private readonly selectedLogId = signal<number | null>(null);
  private readonly highlightedLogIds = signal<Set<number>>(new Set<number>());
  /**
   * Computes all timelines available in the current inspection data.
   */
  private readonly filteredTimelines = computed<
    ReadonlyDomainElement<Timeline[]>
  >(() => this.inspectionDataStore.timelineView()?.filteredTimelines() ?? []);

  /**
   * Holds the currently selected timeline.
   */
  public readonly selectedTimeline =
    signal<ReadonlyDomainElement<Timeline> | null>(null);

  /**
   * Holds the currently highlighted timeline.
   */
  public readonly highlightedTimeline =
    signal<ReadonlyDomainElement<Timeline> | null>(null);

  /**
   * Holds the currently selected revision.
   */
  public readonly selectedRevision =
    signal<ReadonlyDomainElement<Revision> | null>(null);

  /**
   * Indicates whether child timelines should be included in the timeline selection.
   */
  public readonly timelineSelectionShouldIncludeChildren =
    signal<boolean>(true);

  // Derived computed signals.

  /**
   * Computes the currently selected log based on the selection ID.
   */
  public readonly selectedLog = computed<ReadonlyDomainElement<Log> | null>(
    () => {
      const id = this.selectedLogId();
      if (id === null) return null;
      const data = this.inspectionDataStore.inspectionData();
      if (!data) return null;
      try {
        return data.logStore.getLog(id);
      } catch {
        return null;
      }
    },
  );

  /**
   * Computes the list of currently highlighted logs based on the highlight IDs.
   */
  public readonly highlightedLogs = computed<ReadonlyDomainElement<Log>[]>(
    () => {
      const data = this.inspectionDataStore.inspectionData();
      if (!data) return [];
      const ids = this.highlightedLogIds();
      const result: ReadonlyDomainElement<Log>[] = [];
      for (const id of ids) {
        try {
          result.push(data.logStore.getLog(id));
        } catch {
          // Ignore invalid IDs
        }
      }
      return result;
    },
  );

  /**
   * Computes the index of the currently selected log.
   * Returns -1 if no log is selected.
   */
  public readonly selectedLogIndex = computed<number>(() => {
    const l = this.selectedLog();
    return l !== null ? l.logIndex : -1;
  });

  /**
   * Computes a set of indices of the currently highlighted logs.
   */
  public readonly highlightLogIndices = computed<Set<number>>(() => {
    const data = this.inspectionDataStore.inspectionData();
    if (!data) return new Set();
    const ids = this.highlightedLogIds();
    const indices = new Set<number>();
    for (const id of ids) {
      try {
        indices.add(data.logStore.getIndex(id));
      } catch {
        // Ignore
      }
    }
    return indices;
  });

  /**
   * Computes the revisions on the current highlighted timeline that correspond to highlighted logs.
   */
  public readonly highlightedRevisionsOnCurrentTimeline = computed<
    ReadonlyDomainElement<Revision>[]
  >(() => {
    const timeline = this.highlightedTimeline();
    const logIndices = this.highlightLogIndices();
    const result: ReadonlyDomainElement<Revision>[] = [];
    if (timeline === null) return result;
    for (const revision of timeline.revisions) {
      if (logIndices.has(revision.logIndex)) {
        result.push(revision);
      }
    }
    return result;
  });

  /**
   * Computes the revision that precedes the currently selected revision on the same timeline.
   */
  public readonly previousOfSelectedRevision =
    computed<ReadonlyDomainElement<Revision> | null>(() => {
      const revision = this.selectedRevision();
      const timeline = this.selectedTimeline();
      if (revision === null || timeline === null) return null;
      const revisionIndex = timeline.revisions.indexOf(revision);
      return revisionIndex > 0 ? timeline.revisions[revisionIndex - 1] : null;
    });

  /**
   * Computes the list of selected timelines including their descendants if configured.
   */
  public readonly selectedTimelinesWithChildren = computed<
    ReadonlyDomainElement<Timeline>[]
  >(() => {
    const selectedTimeline = this.selectedTimeline();
    const shouldIncludeChild = this.timelineSelectionShouldIncludeChildren();
    if (!selectedTimeline) return [];
    if (!shouldIncludeChild) return [selectedTimeline];
    return [selectedTimeline, ...selectedTimeline.descendants()];
  });

  /**
   * Computes the set of descendant timelines of the selected timeline that should be highlighted.
   */
  public readonly highlightedChildrenOfSelectedTimeline = computed<
    Set<ReadonlyDomainElement<Timeline>>
  >(() => {
    const selectedTimeline = this.selectedTimeline();
    const includeChildren = this.timelineSelectionShouldIncludeChildren();
    const allTimelines = this.filteredTimelines();
    if (!includeChildren || selectedTimeline === null) return new Set();
    for (const timeline of allTimelines) {
      if (timeline === selectedTimeline) {
        return new Set(timeline.descendants());
      }
    }
    return new Set();
  });

  /**
   * Handles selection of a timeline.
   * If the timeline is missing or null, the selection is cleared.
   */
  public onSelectTimeline(timeline?: ReadonlyDomainElement<Timeline> | null) {
    this.selectedTimeline.set(timeline ?? null);
    this.synchronizeSelection();
  }

  /**
   * Handles highlighting of a timeline.
   * If the timeline is missing or null, the highlight is cleared.
   */
  public onHighlightTimeline(
    timeline?: ReadonlyDomainElement<Timeline> | null,
  ) {
    this.highlightedTimeline.set(timeline ?? null);
  }

  /**
   * Highlights specific log entries.
   */
  public onHighlightLog(...logs: ReadonlyDomainElement<Log>[]) {
    this.highlightedLogIds.set(new Set(logs.map((log) => log.id)));
  }

  /**
   * Selects a log entry and updates dependent selections.
   */
  public onSelectLog(log: ReadonlyDomainElement<Log> | null) {
    this.changeSelectionByLogInternal(log, false);
    this.synchronizeSelection();
  }

  /**
   * Selects an event and updates timeline and log selections.
   */
  public onSelectEvent(event: ReadonlyDomainElement<Event>) {
    this.changeSelectionByEventInternal(event.timeline, event, false);
    this.synchronizeSelection();
  }

  /**
   * Selects a revision and updates timeline and log selections.
   */
  public onSelectRevision(revision: ReadonlyDomainElement<Revision> | null) {
    if (revision === null) {
      this.selectedRevision.set(null);
      this.synchronizeSelection();
      return;
    }
    this.changeSelectionByRevisionInternal(revision.timeline, revision, false);
    this.synchronizeSelection();
  }

  /**
   * Synchronizes and validates selection status when any dependent signals change.
   *
   * Clears the log or revision selection if they are not part of the currently selected timeline.
   */
  private synchronizeSelection() {
    const timeline = this.selectedTimeline();
    if (!timeline) return;

    const log = this.selectedLog();
    const revision = this.selectedRevision();

    if (
      log &&
      timeline.lookupEventFromLog(log) === null &&
      timeline.lookupRevisionFromLog(log) === null
    ) {
      // Clear log selection if it is not part of the newly selected timeline.
      this.onSelectLog(null);
    }

    if (revision && timeline.lookupRevisionFromLog(revision.log) === null) {
      this.selectedRevision.set(null);
    }
  }

  private changeSelectionByLogInternal(
    log: ReadonlyDomainElement<Log> | null,
    ignoreResourceSelect: boolean,
  ) {
    this.selectedLogId.set(log ? log.id : null);
    if (ignoreResourceSelect || !log) return;

    const timelines = this.filteredTimelines();

    // Find the timeline and the matching revision or event in a single pass using binary search
    let relatedRevision: ReadonlyDomainElement<Revision> | null = null;
    let relatedEvent: ReadonlyDomainElement<Event> | null = null;
    let targetTimeline: ReadonlyDomainElement<Timeline> | null = null;
    for (const timeline of timelines) {
      relatedRevision = timeline.lookupRevisionFromLog(log);
      if (relatedRevision !== null) {
        targetTimeline = timeline;
        break;
      }
      relatedEvent = timeline.lookupEventFromLog(log);
      if (relatedEvent !== null) {
        targetTimeline = timeline;
        break;
      }
    }
    if (!targetTimeline) return;

    if (relatedRevision) {
      this.changeSelectionByRevisionInternal(
        targetTimeline,
        relatedRevision,
        true,
        true,
      );
      return;
    }

    if (relatedEvent) {
      this.changeSelectionByEventInternal(
        targetTimeline,
        relatedEvent,
        true,
        true,
      );
      return;
    }
  }

  private changeSelectionByEventInternal(
    timeline: ReadonlyDomainElement<Timeline>,
    event: ReadonlyDomainElement<Event>,
    ignoreLogSelect: boolean,
    ignoreTimelineSelect: boolean = false,
  ) {
    if (!ignoreTimelineSelect) this.selectedTimeline.set(timeline);
    if (!ignoreLogSelect) this.changeSelectionByLogInternal(event.log, true);
  }

  private changeSelectionByRevisionInternal(
    timeline: ReadonlyDomainElement<Timeline>,
    revision: ReadonlyDomainElement<Revision>,
    ignoreLogSelect: boolean,
    ignoreTimelineSelect: boolean = false,
  ) {
    this.selectedRevision.set(revision);
    if (!ignoreTimelineSelect) this.selectedTimeline.set(timeline);
    if (!ignoreLogSelect) this.changeSelectionByLogInternal(revision.log, true);
  }
}
