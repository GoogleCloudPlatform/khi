/**
 * Copyright 2024 Google LLC
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
  Component,
  OnDestroy,
  computed,
  inject,
  output,
  signal,
  effect,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { BreakpointObserver } from '@angular/cdk/layout';
import { map } from 'rxjs';
import { ViewStateService } from '../services/view-state.service';
import { ToolbarComponent } from './components/toolbar.component';
import { Subject } from 'rxjs';
import { SelectionManagerV2 } from '../services/selection-manager-v2.service';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import {
  CelTimelineFilter,
  CelLogFilter,
} from 'src/app/store/domain/filter/cel-filter';
import { TimelineFilterConfig } from 'src/app/timeline-toolbar/types/filter-config';
import { TimelineType } from 'src/app/store/domain/style';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';
import { ExcludeNoLogsFilter } from '../store/domain/filter/other-filter';

@Component({
  selector: 'khi-timeline-toolbar-smart',
  templateUrl: './timeline-toolbar-smart.component.html',
  imports: [ToolbarComponent],
})
export class TimelineToolbarSmartComponent implements OnDestroy {
  private readonly viewStateService = inject(ViewStateService);
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);
  private readonly celTimelineFilter = inject(CelTimelineFilter);
  private readonly celLogFilter = inject(CelLogFilter);
  private readonly excludeNoLogsFilter = inject(ExcludeNoLogsFilter);

  /**
   * Direct severity filter option state.
   */
  protected readonly selectedSeverity =
    this.viewStateService.standardSelectedSeverity;

  /**
   * Direct log search filter text state.
   */
  protected readonly logSearchQuery =
    this.viewStateService.standardLogSearchQuery;

  /**
   * Emits an event to switch to advanced mode.
   */
  readonly switchToAdvanced = output<void>();
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly breakpointObserver = inject(BreakpointObserver);

  /**
   * Active timeline filters list.
   */
  protected readonly timelineFilters =
    this.viewStateService.standardTimelineFilters;

  /**
   * The currently selected timeline type for the interactive filter builder.
   */
  protected readonly selectedTimelineTypeForBuilder = signal<string>('*');

  private readonly inspectionData = computed(() => {
    return this.inspectionDataStore.inspectionData();
  });

  /**
   * List of unique timeline types found in the inspection data.
   */
  protected readonly timelineTypes = computed<TimelineType[]>(() => {
    const store = this.inspectionData()?.timelineStore;
    const styleStore = this.inspectionData()?.styleStore;
    if (!store || !styleStore) {
      return [];
    }
    const activeLabels = new Set<string>();
    for (const t of store.timelines) {
      if (t.type?.label) {
        activeLabels.add(t.type.label.toLowerCase());
      }
    }
    return styleStore.timelineTypes
      .filter((t) => activeLabels.has(t.label.toLowerCase()))
      .sort((a, b) => a.label.localeCompare(b.label));
  });

  /**
   * List of candidates for the selected timeline type in the filter builder.
   */
  protected readonly candidates = computed<string[]>(() => {
    const store = this.inspectionData()?.timelineStore;
    const selectedType = this.selectedTimelineTypeForBuilder();
    if (!store || !selectedType || selectedType === '*') {
      return [];
    }
    const candSet = new Set<string>();
    const selectedTypeLower = selectedType.toLowerCase();
    for (const t of store.timelines) {
      for (const node of t.path) {
        if (node.type.label.toLowerCase() === selectedTypeLower) {
          candSet.add(node.label);
        }
      }
    }
    return Array.from(candSet).sort();
  });

  /**
   * Map of lowercased timeline type labels to their configured material icon strings.
   */
  protected readonly typeIcons = computed<Record<string, string>>(() => {
    const styleStore = this.inspectionData()?.styleStore;
    if (!styleStore) {
      return {};
    }
    const map: Record<string, string> = {};
    for (const type of styleStore.timelineTypes) {
      map[type.label.toLowerCase()] = type.icon;
    }
    return map;
  });

  /**
   * Map of lowercased timeline type labels to their computed CSS colors.
   */
  protected readonly typeColors = computed<
    Record<string, { backgroundColor: string; foregroundColor: string }>
  >(() => {
    const styleStore = this.inspectionData()?.styleStore;
    if (!styleStore) {
      return {};
    }
    const map: Record<
      string,
      { backgroundColor: string; foregroundColor: string }
    > = {};
    for (const type of styleStore.timelineTypes) {
      map[type.label.toLowerCase()] = {
        backgroundColor: RendererConvertUtil.hdrColorToCSSColor([
          type.backgroundColor.r,
          type.backgroundColor.g,
          type.backgroundColor.b,
          type.backgroundColor.a,
        ]),
        foregroundColor: RendererConvertUtil.hdrColorToCSSColor([
          type.foregroundColor.r,
          type.foregroundColor.g,
          type.foregroundColor.b,
          type.foregroundColor.a,
        ]),
      };
    }
    return map;
  });

  /**
   * Map of lowercased timeline type labels to their total candidate count.
   */
  protected readonly typeCandidateCounts = computed<Record<string, number>>(
    () => {
      const store = this.inspectionData()?.timelineStore;
      if (!store) {
        return {};
      }
      const counts: Record<string, Set<string>> = {};
      for (const t of store.timelines) {
        for (const node of t.path) {
          const labelLower = node.type.label.toLowerCase();
          if (!counts[labelLower]) {
            counts[labelLower] = new Set<string>();
          }
          counts[labelLower].add(node.label);
        }
      }
      const result: Record<string, number> = {};
      for (const key of Object.keys(counts)) {
        result[key] = counts[key].size;
      }
      return result;
    },
  );

  /**
   * An empty set used as a fallback for template bindings.
   */
  protected readonly emptySet = new Set<string>();

  /**
   * Signal indicating whether to show button labels based on screen width.
   */
  protected readonly showButtonLabel = toSignal(
    this.breakpointObserver
      .observe(['(min-width: 1200px)'])
      .pipe(map((result) => result.matches)),
  );

  /**
   * Signal containing the current timezone shift in hours.
   */
  protected readonly timezoneShift = toSignal(
    this.viewStateService.timezoneShift,
  );

  /**
   * Signal indicating if no log or timeline is selected.
   */
  protected readonly logOrTimelineNotSelected = computed(() => {
    const selectedTimeline = this.selectionManager.selectedTimeline();
    const selectedLog = this.selectionManager.selectedLog();
    return selectedTimeline == null || selectedLog == null;
  });

  private readonly destroyed = new Subject<void>();

  constructor() {
    const viewState = this.viewStateService;
    const currentTimelineCel = this.celTimelineFilter.celExpr();
    const currentLogCel = this.celLogFilter.celExpr();

    const hypotheticalTimelineCel = this.compileFiltersToCel(
      viewState.standardTimelineFilters(),
    );
    const hypotheticalLogCel = this.compileLogFiltersToCel(
      viewState.standardSelectedSeverity(),
      viewState.standardLogSearchQuery(),
    );

    if (currentTimelineCel !== hypotheticalTimelineCel) {
      viewState.standardTimelineFilters.set([]);
      this.celTimelineFilter.updateFilter('');
    }

    if (currentLogCel !== hypotheticalLogCel) {
      viewState.standardSelectedSeverity.set('ANY');
      viewState.standardLogSearchQuery.set('');
      this.celLogFilter.updateFilter('');
    }

    effect(() => {
      const filters = this.timelineFilters();
      const celExpr = this.compileFiltersToCel(filters);
      this.celTimelineFilter.updateFilter(celExpr);
    });

    effect(() => {
      const severity = this.selectedSeverity();
      const searchQuery = this.logSearchQuery();
      const celExpr = this.compileLogFiltersToCel(severity, searchQuery);
      this.celLogFilter.updateFilter(celExpr);
    });
    this.excludeNoLogsFilter.enabled.set(true);
  }

  private compileLogFiltersToCel(
    severity: string,
    searchQuery: string,
  ): string {
    const parts: string[] = [];
    if (severity && severity !== 'ANY') {
      parts.push(`severity >= ${severity}`);
    }
    if (searchQuery && searchQuery.trim() !== '') {
      const escaped = searchQuery.replace(/"/g, '\\"');
      parts.push(`body("${escaped}")`);
    }
    return parts.join(' && ');
  }

  private compileFiltersToCel(filters: TimelineFilterConfig[]): string {
    if (filters.length === 0) {
      return '';
    }
    return filters
      .map((f) => {
        let celValue = f.value.replace(/"/g, '\\"');
        if (f.mode === 'selection') {
          const escapedParts = f.value
            .split('|')
            .map((val) => val.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
            .map((val) => val.replace(/"/g, '\\"'));
          celValue = `^(?:${escapedParts.join('|')})$`;
        }
        if (f.timelineType === '*') {
          return `match("${celValue}")`;
        } else {
          return `match("${f.timelineType}", "${celValue}")`;
        }
      })
      .join(' && ');
  }

  ngOnDestroy() {
    this.destroyed.next();
    this.destroyed.complete();
  }

  /**
   * Handles the commit of a new timezone shift value.
   */
  protected onTimezoneshiftCommit(value: number) {
    this.viewStateService.setTimezoneShift(value);
  }

  /**
   * Opens the graph page in a new tab.
   */
  protected onDrawDiagram() {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
