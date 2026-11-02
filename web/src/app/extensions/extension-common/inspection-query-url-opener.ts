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

import { EffectRef, effect, inject, signal } from '@angular/core';
import { URLDataOpenerExtension } from 'src/app/extensions/extension-common/extension-types/url-data-opener';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { ViewStateService } from 'src/app/services/view-state.service';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Opens an existing backend inspection from /session/:id query parameters and synchronizes active selection and filter state back to the URL.
 */
export class InspectionQueryUrlOpener implements URLDataOpenerExtension {
  private readonly getLocationSearch: () => string;
  private readonly replaceLocationSearch: (search: string) => void;
  private readonly isRestoringFromUrl = signal<boolean>(false);
  private syncEffectRef: EffectRef | null = null;

  constructor(
    getLocationSearch: () => string = () => window.location.search,
    replaceLocationSearch: (
      search: string,
    ) => void = defaultReplaceLocationSearch,
  ) {
    this.getLocationSearch = getLocationSearch;
    this.replaceLocationSearch = replaceLocationSearch;
  }

  /**
   * Checks URL query parameters for inspectionId, initiates loading and selection when present, and registers reactive URL synchronization.
   */
  public tryOpen(): boolean {
    const dataLoader = inject(InspectionDataLoaderService);
    const dataStore = inject(InspectionDataStore);
    const selectionManager = inject(SelectionManager);
    const viewState = inject(ViewStateService);

    this.registerUrlSync(dataStore, selectionManager);

    const params = new URLSearchParams(this.getLocationSearch());
    const inspectionId = params.get('inspectionId')?.trim();
    if (!inspectionId) {
      return false;
    }

    this.isRestoringFromUrl.set(true);
    void this.loadAndSelect(
      inspectionId,
      params,
      dataLoader,
      dataStore,
      selectionManager,
      viewState,
    );
    return true;
  }

  private registerUrlSync(
    dataStore: InspectionDataStore,
    selectionManager: SelectionManager,
  ): void {
    if (this.syncEffectRef !== null) {
      return;
    }
    this.syncEffectRef = effect(() => {
      this.syncUrlFromState(dataStore, selectionManager);
    });
  }

  private syncUrlFromState(
    dataStore: InspectionDataStore,
    selectionManager: SelectionManager,
  ): void {
    const data = dataStore.inspectionData();
    const inspectionId = dataStore.inspectionId()?.trim() ?? '';
    const timelineView = dataStore.timelineView();
    const timelineQuery = timelineView?.timelineQuery() ?? '';
    const logQuery = timelineView?.logQuery() ?? '';
    const selectedRevision = selectionManager.selectedRevision();
    const selectedLog = selectionManager.selectedLog();
    const selectedTimeline = selectionManager.selectedTimeline();

    // Suppress URL writes while initial query parameters are still being restored.
    if (this.isRestoringFromUrl() || !data || !inspectionId) {
      return;
    }

    const params = new URLSearchParams(this.getLocationSearch());
    params.set('inspectionId', inspectionId);
    setOrDeleteParam(params, 'timelineQuery', timelineQuery);
    setOrDeleteParam(params, 'logQuery', logQuery);

    const validTimeline =
      selectedTimeline !== null &&
      selectedTimeline.id > 0 &&
      hasTimeline(data, selectedTimeline)
        ? selectedTimeline
        : null;
    const validRevision =
      selectedRevision !== null &&
      selectedRevision.timelineId > 0 &&
      selectedRevision.index >= 0 &&
      hasRevision(data, selectedRevision.timelineId, selectedRevision.index)
        ? selectedRevision
        : null;

    if (
      validRevision !== null &&
      (validTimeline === null || validTimeline.id === validRevision.timelineId)
    ) {
      params.set('timelineId', String(validRevision.timelineId));
      params.set('revisionIndex', String(validRevision.index));
      params.delete('logId');
    } else if (selectedLog !== null && selectedLog.id >= 0) {
      if (validTimeline !== null) {
        params.set('timelineId', String(validTimeline.id));
      } else {
        params.delete('timelineId');
      }
      params.delete('revisionIndex');
      params.set('logId', String(selectedLog.id));
    } else if (validTimeline !== null) {
      params.set('timelineId', String(validTimeline.id));
      params.delete('revisionIndex');
      params.delete('logId');
    } else {
      params.delete('timelineId');
      params.delete('revisionIndex');
      params.delete('logId');
    }

    const encoded = params.toString();
    this.replaceLocationSearch(encoded ? `?${encoded}` : '');
  }

  private async loadAndSelect(
    inspectionId: string,
    params: URLSearchParams,
    dataLoader: InspectionDataLoaderService,
    dataStore: InspectionDataStore,
    selectionManager: SelectionManager,
    viewState: ViewStateService,
  ): Promise<void> {
    try {
      await dataLoader.loadInspectionDataFromBackend(inspectionId);
      const data = dataStore.inspectionData();
      if (!data) {
        return;
      }

      const timelineView = dataStore.timelineView();
      if (timelineView) {
        timelineView.expandAllTimelines();
        const timelineQuery = params.get('timelineQuery') ?? '';
        const logQuery = params.get('logQuery') ?? '';
        if (timelineQuery || logQuery) {
          viewState.isAdvancedMode.set(true);
          viewState.advancedTimelineIncludeCel.set(timelineQuery);
          viewState.advancedLogCel.set(logQuery);
          timelineView.backendFilter.updateFilterParams({
            timelineQuery,
            logQuery,
          });
        }
      }

      const timelineId = parseNonNegativeInt(params.get('timelineId'));
      const revisionIndex = parseNonNegativeInt(params.get('revisionIndex'));
      const logId = parseNonNegativeInt(params.get('logId'));

      let targetTimeline: ReadonlyDomainElement<Timeline> | null = null;
      if (timelineId !== null && timelineId > 0) {
        try {
          targetTimeline = data.timelineStore.getTimeline(timelineId);
        } catch {
          targetTimeline = null;
        }
      }

      if (
        targetTimeline &&
        revisionIndex !== null &&
        revisionIndex < targetTimeline.revisions.length
      ) {
        selectionManager.onSelectRevision(
          targetTimeline.revisions[revisionIndex],
        );
        return;
      }

      if (logId !== null) {
        if (targetTimeline) {
          selectionManager.onSelectTimeline(targetTimeline);
        }
        try {
          const log = data.logStore.getLog(logId);
          await selectionManager.onSelectLog(log);
          return;
        } catch {
          // Fall through to timeline selection if logId is out of range.
        }
      }

      if (targetTimeline) {
        selectionManager.onSelectTimeline(targetTimeline);
      }
    } finally {
      this.isRestoringFromUrl.set(false);
    }
  }
}

function setOrDeleteParam(
  params: URLSearchParams,
  key: string,
  value: string,
): void {
  if (value !== '') {
    params.set(key, value);
  } else {
    params.delete(key);
  }
}

function hasTimeline(
  data: NonNullable<ReturnType<InspectionDataStore['inspectionData']>>,
  timeline: ReadonlyDomainElement<Timeline>,
): boolean {
  try {
    return data.timelineStore.getTimeline(timeline.id) === timeline;
  } catch {
    return false;
  }
}

function hasRevision(
  data: NonNullable<ReturnType<InspectionDataStore['inspectionData']>>,
  timelineId: number,
  revisionIndex: number,
): boolean {
  try {
    const timeline = data.timelineStore.getTimeline(timelineId);
    return revisionIndex < timeline.revisions.length;
  } catch {
    return false;
  }
}

function defaultReplaceLocationSearch(search: string): void {
  const pathname = window.location.pathname;
  const hash = window.location.hash;
  const normalizedSearch = search
    ? search.startsWith('?')
      ? search
      : `?${search}`
    : '';
  const nextUrl = `${pathname}${normalizedSearch}${hash}`;
  const currentUrl = `${pathname}${window.location.search}${hash}`;
  if (nextUrl !== currentUrl) {
    window.history.replaceState(window.history.state, '', nextUrl);
  }
}

function parseNonNegativeInt(raw: string | null): number | null {
  if (raw === null) {
    return null;
  }
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isSafeInteger(parsed) ? parsed : null;
}
