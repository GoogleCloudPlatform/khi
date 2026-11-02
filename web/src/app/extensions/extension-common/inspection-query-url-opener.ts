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

import { inject } from '@angular/core';
import { URLDataOpenerExtension } from 'src/app/extensions/extension-common/extension-types/url-data-opener';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Opens an existing backend inspection and focuses a target timeline, revision, or log from /session/:id query parameters.
 */
export class InspectionQueryUrlOpener implements URLDataOpenerExtension {
  private readonly getLocationSearch: () => string;

  constructor(getLocationSearch: () => string = () => window.location.search) {
    this.getLocationSearch = getLocationSearch;
  }

  /**
   * Checks URL query parameters for inspectionId and initiates loading and selection when present.
   */
  public tryOpen(): boolean {
    const params = new URLSearchParams(this.getLocationSearch());
    const inspectionId = params.get('inspectionId')?.trim();
    if (!inspectionId) {
      return false;
    }

    const dataLoader = inject(InspectionDataLoaderService);
    const dataStore = inject(InspectionDataStore);
    const selectionManager = inject(SelectionManager);

    void this.loadAndSelect(
      inspectionId,
      params,
      dataLoader,
      dataStore,
      selectionManager,
    );
    return true;
  }

  private async loadAndSelect(
    inspectionId: string,
    params: URLSearchParams,
    dataLoader: InspectionDataLoaderService,
    dataStore: InspectionDataStore,
    selectionManager: SelectionManager,
  ): Promise<void> {
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
