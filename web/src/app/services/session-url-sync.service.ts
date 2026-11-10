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
  DestroyRef,
  Injectable,
  InjectionToken,
  Injector,
  effect,
  inject,
  signal,
} from '@angular/core';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { ViewStateService } from 'src/app/services/view-state.service';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineView } from 'src/app/store/domain/timeline-view';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Adapter interface for reading and replacing the browser URL search query without triggering router navigation.
 */
export interface SessionLocationAdapter {
  /**
   * Returns the current URL search string, including the leading '?' when non-empty.
   */
  readonly getSearch: () => string;
  /**
   * Replaces the current URL search string in browser history.
   */
  readonly replaceSearch: (search: string) => void;
}

/**
 * Dependency injection token providing browser URL search access for {@link SessionUrlSyncService}.
 */
export const SESSION_LOCATION_ADAPTER =
  new InjectionToken<SessionLocationAdapter>('SESSION_LOCATION_ADAPTER', {
    providedIn: 'root',
    factory: () => ({
      getSearch: () => window.location.search,
      replaceSearch: defaultReplaceLocationSearch,
    }),
  });

/**
 * Opens an existing backend inspection from /session/:id query parameters and synchronizes active selection and filter state back to the URL.
 */
@Injectable()
export class SessionUrlSyncService {
  private readonly injector = inject(Injector);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dataLoader = inject(InspectionDataLoaderService);
  private readonly dataStore = inject(InspectionDataStore);
  private readonly selectionManager = inject(SelectionManager);
  private readonly viewState = inject(ViewStateService);
  private readonly locationAdapter = inject(SESSION_LOCATION_ADAPTER);
  private readonly isRestoringFromUrl = signal<boolean>(false);

  constructor() {
    effect(() => {
      this.syncUrlFromState();
    });
  }

  /**
   * Checks URL query parameters for an inspectionId and initiates loading and target selection when present.
   *
   * @returns True when an inspection load was initiated from URL query parameters.
   */
  public openFromQueryParams(): boolean {
    const params = new URLSearchParams(this.locationAdapter.getSearch());
    const inspectionId = params.get('inspectionId')?.trim();
    if (!inspectionId) {
      return false;
    }

    this.isRestoringFromUrl.set(true);
    void this.loadAndSelect(inspectionId, params);
    return true;
  }

  private syncUrlFromState(): void {
    const data = this.dataStore.inspectionData();
    const inspectionId = this.dataStore.inspectionId()?.trim() ?? '';
    const timelineView = this.dataStore.timelineView();
    const timelineQuery = timelineView?.timelineQuery() ?? '';
    const logQuery = timelineView?.logQuery() ?? '';
    const selectedRevision = this.selectionManager.selectedRevision();
    const selectedLog = this.selectionManager.selectedLog();
    const selectedTimeline = this.selectionManager.selectedTimeline();

    // Suppresses URL writes while initial query parameters are still being restored.
    if (this.isRestoringFromUrl() || !data || !inspectionId) {
      return;
    }

    const params = new URLSearchParams(this.locationAdapter.getSearch());
    params.set('inspectionId', inspectionId);
    setOrDeleteParam(params, 'timelineQuery', timelineQuery);
    setOrDeleteParam(params, 'logQuery', logQuery);

    const timelineId = selectedRevision?.timelineId ?? selectedTimeline?.id;
    const revisionIndex = selectedRevision?.index;
    const logId = selectedRevision === null ? selectedLog?.id : undefined;
    setOrDeleteParam(params, 'timelineId', String(timelineId ?? ''));
    setOrDeleteParam(params, 'revisionIndex', String(revisionIndex ?? ''));
    setOrDeleteParam(params, 'logId', String(logId ?? ''));

    const encoded = params.toString();
    this.locationAdapter.replaceSearch(encoded ? `?${encoded}` : '');
  }

  private async loadAndSelect(
    inspectionId: string,
    params: URLSearchParams,
  ): Promise<void> {
    try {
      await this.dataLoader.loadInspectionDataFromBackend(inspectionId);
      const data = this.dataStore.inspectionData();
      if (!data) {
        return;
      }

      const timelineView = this.dataStore.timelineView();
      if (timelineView) {
        timelineView.expandAllTimelines();
        const timelineQuery = params.get('timelineQuery') ?? '';
        const logQuery = params.get('logQuery') ?? '';
        if (timelineQuery || logQuery) {
          this.viewState.isAdvancedMode.set(true);
          this.viewState.advancedTimelineIncludeCel.set(timelineQuery);
          this.viewState.advancedLogCel.set(logQuery);
          timelineView.backendFilter.updateFilterParams({
            timelineQuery,
            logQuery,
          });
        }
        await this.waitForFilteringComplete(timelineView);
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
        this.selectionManager.onSelectRevision(
          targetTimeline.revisions[revisionIndex],
        );
        return;
      }

      if (logId !== null) {
        if (targetTimeline) {
          this.selectionManager.onSelectTimeline(targetTimeline);
        }
        try {
          const log = data.logStore.getLog(logId);
          await this.selectionManager.onSelectLog(log);
          return;
        } catch {
          // Falls through to timeline selection when logId is out of range.
        }
      }

      if (targetTimeline) {
        this.selectionManager.onSelectTimeline(targetTimeline);
      }
    } finally {
      this.isRestoringFromUrl.set(false);
    }
  }

  private async waitForFilteringComplete(
    timelineView: TimelineView,
  ): Promise<void> {
    await Promise.resolve();
    if (!timelineView.isFiltering()) {
      return;
    }
    return new Promise<void>((resolve) => {
      const ref = effect(
        () => {
          if (!timelineView.isFiltering()) {
            ref.destroy();
            resolve();
          }
        },
        { injector: this.injector },
      );
      this.destroyRef.onDestroy(() => {
        ref.destroy();
        resolve();
      });
    });
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
