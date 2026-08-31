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
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { Log } from 'src/app/store/domain/log';
import { Timeline, Revision, Event } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * SelectionManager provides selected/highlighted list of logs, timelines, revisions or events from the received user interaction.
 * Signal-based modern version of SelectionManagerService.
 */
@Injectable({ providedIn: 'root' })
export class SelectionManager {
  private readonly inspectionDataStore = inject(InspectionDataStore);

  // Writable signals representing internal state.
  private readonly selectedLogId = signal<number | null>(null);
  private readonly highlightedLogIds = signal<Set<number>>(new Set<number>());

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
      if (revision === null) return null;
      return revision.prev;
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
    if (!includeChildren || selectedTimeline === null) return new Set();
    return new Set(selectedTimeline.descendants());
  });

  /**
   * Handles selection of a timeline.
   * If the timeline is missing or null, the selection is cleared.
   */
  public onSelectTimeline(timeline?: ReadonlyDomainElement<Timeline> | null) {
    const nextTimeline = timeline ?? null;
    this.selectedTimeline.set(nextTimeline);
    this.validateSelectionAgainstTimeline(nextTimeline);
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
    if (!log) {
      this.selectedLogId.set(null);
      this.selectedRevision.set(null);
      return;
    }

    this.selectedLogId.set(log.id);

    const timelineStore =
      this.inspectionDataStore.inspectionData()?.timelineStore;
    if (!timelineStore) {
      this.selectedRevision.set(null);
      return;
    }

    const logTimelines = timelineStore.getTimelinesForLogId(log.id);
    if (logTimelines.length === 0) {
      this.selectedRevision.set(null);
      return;
    }

    const filteredTimelineIds = this.inspectionDataStore
      .timelineView()
      ?.filteredTimelineIds();

    const selectedTimeline = this.selectedTimeline();
    const currentTimelines = this.selectedTimelinesWithChildren();

    // Priority 1: Check the main selected timeline first if it is visible.
    if (
      selectedTimeline &&
      (!filteredTimelineIds || filteredTimelineIds.has(selectedTimeline.id))
    ) {
      const revision = selectedTimeline.lookupRevisionFromLog(log);
      if (revision) {
        this.selectedRevision.set(revision);
        return;
      }
      const event = selectedTimeline.lookupEventFromLog(log);
      if (event) {
        this.selectedRevision.set(null);
        return;
      }
    }

    // Priority 2: Check descendant timelines of the selected timeline (if visible).
    if (currentTimelines.length > 1) {
      for (const timeline of currentTimelines) {
        if (timeline === selectedTimeline) continue;
        if (filteredTimelineIds && !filteredTimelineIds.has(timeline.id)) {
          continue;
        }
        const revision = timeline.lookupRevisionFromLog(log);
        if (revision) {
          this.selectedRevision.set(revision);
          return;
        }
        const event = timeline.lookupEventFromLog(log);
        if (event) {
          this.selectedRevision.set(null);
          return;
        }
      }
    }

    // Priority 3: When no timeline is selected or the log is not found in the currently selected timelines,
    // automatically select the first visible timeline containing the log.
    for (const timeline of logTimelines) {
      if (!filteredTimelineIds || filteredTimelineIds.has(timeline.id)) {
        this.selectedTimeline.set(timeline);
        const revision = timeline.lookupRevisionFromLog(log);
        this.selectedRevision.set(revision);
        return;
      }
    }
    this.selectedRevision.set(null);
  }

  /**
   * Selects an event and updates timeline and log selections.
   */
  public onSelectEvent(event: ReadonlyDomainElement<Event>) {
    this.selectedTimeline.set(event.timeline);
    this.selectedLogId.set(event.log.id);
    this.selectedRevision.set(null);
  }

  /**
   * Selects a revision and updates timeline and log selections.
   */
  public onSelectRevision(revision: ReadonlyDomainElement<Revision> | null) {
    if (revision === null) {
      this.selectedRevision.set(null);
      return;
    }
    this.selectedTimeline.set(revision.timeline);
    this.selectedLogId.set(revision.log.id);
    this.selectedRevision.set(revision);
  }

  /**
   * Validates that the current log/revision selection is within the target timeline or its children.
   */
  private validateSelectionAgainstTimeline(
    timeline: ReadonlyDomainElement<Timeline> | null,
  ) {
    if (!timeline) {
      this.selectedLogId.set(null);
      this.selectedRevision.set(null);
      return;
    }

    const log = this.selectedLog();
    const revision = this.selectedRevision();
    const validTimelines = this.selectedTimelinesWithChildren();
    const validTimelineIds = new Set(validTimelines.map((t) => t.id));

    const isLogValid = this.isSelectedLogValid(
      log,
      validTimelineIds,
      validTimelines,
    );
    const isRevisionValid = this.isSelectedRevisionValid(
      revision,
      validTimelineIds,
    );

    if (!isLogValid) {
      this.selectedLogId.set(null);
      this.selectedRevision.set(null);
    } else if (!isRevisionValid) {
      this.selectedRevision.set(null);
    }
  }

  /**
   * Checks whether the currently selected log is still valid within the active timeline selection.
   */
  private isSelectedLogValid(
    log: ReadonlyDomainElement<Log> | null,
    validTimelineIds: Set<number>,
    validTimelines: ReadonlyDomainElement<Timeline>[],
  ): boolean {
    if (!log) return true;
    const timelineStore =
      this.inspectionDataStore.inspectionData()?.timelineStore;
    if (timelineStore) {
      return timelineStore
        .getTimelineIdsForLogId(log.id)
        .some((id) => validTimelineIds.has(id));
    }
    return validTimelines.some(
      (t) => t.lookupRevisionFromLog(log) || t.lookupEventFromLog(log),
    );
  }

  /**
   * Checks whether the currently selected revision is still valid within the active timeline selection.
   */
  private isSelectedRevisionValid(
    revision: ReadonlyDomainElement<Revision> | null,
    validTimelineIds: Set<number>,
  ): boolean {
    if (!revision) return true;
    return validTimelineIds.has(revision.timelineId);
  }
}
