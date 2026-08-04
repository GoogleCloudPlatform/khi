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
      if (revision === null) return null;
      const timeline = revision.timeline;
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

    const currentTimelines = this.selectedTimelinesWithChildren();

    // When a timeline is already selected, prioritize searching within that timeline and its children.
    if (currentTimelines.length > 0) {
      for (const timeline of currentTimelines) {
        const revision = timeline.lookupRevisionFromLog(log);
        if (revision) {
          this.selectedLogId.set(log.id);
          this.selectedRevision.set(revision);
          return;
        }
        const event = timeline.lookupEventFromLog(log);
        if (event) {
          this.selectedLogId.set(log.id);
          this.selectedRevision.set(null);
          return;
        }
      }
    }

    // When no timeline is selected or the log is not found in the currently selected timelines,
    // automatically select the first timeline containing the log.
    this.selectedLogId.set(log.id);
    for (const timeline of this.filteredTimelines()) {
      const revision = timeline.lookupRevisionFromLog(log);
      if (revision) {
        this.selectedTimeline.set(timeline);
        this.selectedRevision.set(revision);
        return;
      }
      const event = timeline.lookupEventFromLog(log);
      if (event) {
        this.selectedTimeline.set(timeline);
        this.selectedRevision.set(null);
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

    const isLogValid =
      !log ||
      validTimelines.some(
        (t) => t.lookupRevisionFromLog(log) || t.lookupEventFromLog(log),
      );
    const isRevisionValid =
      !revision ||
      validTimelines.some((t) => t.lookupRevisionFromLog(revision.log));

    if (!isLogValid) {
      this.selectedLogId.set(null);
      this.selectedRevision.set(null);
    } else if (!isRevisionValid) {
      this.selectedRevision.set(null);
    }
  }
}
