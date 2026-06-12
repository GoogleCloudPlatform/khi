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
  signal,
  effect,
  untracked,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { BreakpointObserver } from '@angular/cdk/layout';
import {
  BehaviorSubject,
  Subject,
  debounceTime,
  distinctUntilChanged,
  map,
  takeUntil,
} from 'rxjs';
import { ViewStateService } from 'src/app/services/view-state.service';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import {
  CelTimelineFilter,
  CelLogFilter,
} from 'src/app/store/domain/filter/cel-filter';
import { ExcludeNoLogsFilter } from 'src/app/store/domain/filter/other-filter';
import { TimelineFilterConfig } from 'src/app/timeline-toolbar/types/filter-config';
import { TimelineType } from 'src/app/store/domain/style';
import { ToolbarFrameComponent } from './components/toolbar-frame.component';

/**
 * Acts as a single unified smart container logic controller for both standard and advanced timeline toolbars.
 */
@Component({
  selector: 'khi-timeline-toolbar-smart',
  templateUrl: './timeline-toolbar-smart.component.html',
  imports: [ToolbarFrameComponent],
})
export class TimelineToolbarSmartComponent implements OnDestroy {
  private readonly viewStateService = inject(ViewStateService);
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);
  private readonly celTimelineFilter = inject(CelTimelineFilter);
  private readonly celLogFilter = inject(CelLogFilter);
  private readonly excludeNoLogsFilter = inject(ExcludeNoLogsFilter);
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly breakpointObserver = inject(BreakpointObserver);

  private readonly destroyed = new Subject<void>();

  // Global state
  /** Global display mode state signal managed by ViewStateService. */
  protected readonly isAdvancedMode = this.viewStateService.isAdvancedMode;

  /** Signal holding the current timezone shift offset in hours. */
  protected readonly timezoneShift = toSignal(
    this.viewStateService.timezoneShift,
    {
      initialValue: 0,
    },
  );

  /** Whether the app is currently filtering timelines and logs asynchronously. */
  protected readonly isFiltering = computed(() => {
    const view = this.inspectionDataStore.timelineView();
    return view ? view.isFiltering() : false;
  });

  /** Holds the current concrete filtering progress details. */
  private readonly filteringProgress = computed(() => {
    const view = this.inspectionDataStore.timelineView();
    return view ? view.progress() : null;
  });

  /** Computes the percentage completion of the current filtering step. */
  protected readonly progressPercent = computed(() => {
    const p = this.filteringProgress();
    return p ? Math.round((p.current / p.total) * 100) : 0;
  });

  /** Signal locking button triggers when log/timeline selection context is missing. */
  protected readonly logOrTimelineNotSelected = computed(() => {
    const selectedTimeline = this.selectionManager.selectedTimeline();
    const selectedLog = this.selectionManager.selectedLog();
    return selectedTimeline == null || selectedLog == null;
  });

  // Standard Mode properties
  /** Direct severity filter option state. */
  protected readonly selectedSeverity =
    this.viewStateService.standardSelectedSeverity;

  /** Direct log search filter text query. */
  protected readonly logSearchQuery =
    this.viewStateService.standardLogSearchQuery;

  /** Configured active standard timeline filters. */
  protected readonly timelineFilters =
    this.viewStateService.standardTimelineFilters;

  /** Selected timeline type used within interactive filter builders. */
  protected readonly selectedTimelineTypeForBuilder = signal<string>('*');

  private readonly inspectionData = computed(() => {
    return this.inspectionDataStore.inspectionData();
  });

  /** List of unique timeline types located within loaded store elements. */
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

  /** Interactive timeline label suggestions list. */
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

  /** Summary count metrics. */
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

  /** Hides labels if screen viewport width is small. */
  protected readonly showButtonLabel = toSignal(
    this.breakpointObserver
      .observe(['(min-width: 1200px)'])
      .pipe(map((result) => result.matches)),
    { initialValue: true },
  );

  // Advanced Mode properties
  private readonly timelineCelFilter$ = new BehaviorSubject<string>('');
  private readonly logCelFilter$ = new BehaviorSubject<string>('');

  /** Active advanced timeline CEL expression signal. */
  protected readonly timelineCelFilter = toSignal(this.timelineCelFilter$, {
    initialValue: '',
  });

  /** Active advanced log CEL expression signal. */
  protected readonly logCelFilter = toSignal(this.logCelFilter$, {
    initialValue: '',
  });

  /** Validation result error output for timeline CEL queries. */
  protected readonly timelineCelError = toSignal(
    this.timelineCelFilter$.pipe(
      map((val) => {
        if (!val || val.trim() === '') return '';
        const checkRes = this.celTimelineFilter.validate(val);
        return checkRes.success
          ? ''
          : (checkRes.error?.message ?? 'Invalid CEL expression.');
      }),
    ),
    { initialValue: '' },
  );

  /** Validation result error output for log CEL queries. */
  protected readonly logCelError = toSignal(
    this.logCelFilter$.pipe(
      map((val) => {
        if (!val || val.trim() === '') return '';
        const checkRes = this.celLogFilter.validate(val);
        return checkRes.success
          ? ''
          : (checkRes.error?.message ?? 'Invalid CEL expression.');
      }),
    ),
    { initialValue: '' },
  );

  /** Active option toggle matching log hits. */
  protected readonly hideTimelinesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideTimelinesWithoutMatchingLogs,
    { initialValue: true },
  );

  constructor() {
    const viewState = this.viewStateService;
    const currentTimelineCel = this.celTimelineFilter.celExpr();
    const currentLogCel = this.celLogFilter.celExpr();

    const hypotheticalTimelineCel = compileFiltersToCel(
      viewState.standardTimelineFilters(),
    );
    const hypotheticalLogCel = compileLogFiltersToCel(
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

    // Effects executing standard compiler logic
    effect(() => {
      if (!this.isAdvancedMode()) {
        const filters = this.timelineFilters();
        const celExpr = compileFiltersToCel(filters);
        this.celTimelineFilter.updateFilter(celExpr);
      }
    });

    effect(() => {
      if (!this.isAdvancedMode()) {
        const severity = this.selectedSeverity();
        const searchQuery = this.logSearchQuery();
        const celExpr = compileLogFiltersToCel(severity, searchQuery);
        this.celLogFilter.updateFilter(celExpr);
      }
    });

    // Sync advanced mode CEL triggers
    effect(() => {
      if (this.isAdvancedMode()) {
        const currentTimelineExpr = this.celTimelineFilter.celExpr();
        if (this.timelineCelFilter$.value !== currentTimelineExpr) {
          const hasError = untracked(() => this.timelineCelError() !== '');
          if (currentTimelineExpr !== '' || !hasError) {
            this.timelineCelFilter$.next(currentTimelineExpr);
          }
        }
      }
    });

    effect(() => {
      if (this.isAdvancedMode()) {
        const currentLogExpr = this.celLogFilter.celExpr();
        if (this.logCelFilter$.value !== currentLogExpr) {
          const hasError = untracked(() => this.logCelError() !== '');
          if (currentLogExpr !== '' || !hasError) {
            this.logCelFilter$.next(currentLogExpr);
          }
        }
      }
    });

    // Advanced mode RxJS streams debouncers
    this.timelineCelFilter$
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        takeUntil(this.destroyed),
      )
      .subscribe((filter) => {
        if (this.isAdvancedMode()) {
          this.celTimelineFilter.updateFilter(filter);
        }
      });

    this.logCelFilter$
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        takeUntil(this.destroyed),
      )
      .subscribe((filter) => {
        if (this.isAdvancedMode()) {
          this.celLogFilter.updateFilter(filter);
        }
      });

    this.viewStateService.hideTimelinesWithoutMatchingLogs
      .pipe(takeUntil(this.destroyed))
      .subscribe((hide) => {
        this.excludeNoLogsFilter.setEnabled(hide);
      });

    this.excludeNoLogsFilter.setEnabled(
      this.hideTimelinesWithoutMatchingLogs(),
    );
  }

  ngOnDestroy() {
    this.destroyed.next();
    this.destroyed.complete();
  }

  /**
   * Handles the commit of a timezone shift offset value.
   */
  protected onTimezoneshiftCommit(value: number) {
    this.viewStateService.setTimezoneShift(value);
  }

  /**
   * Commits a modified timeline CEL filter expression text queries.
   */
  protected onTimelineCelFilterChange(filter: string): void {
    this.timelineCelFilter$.next(filter);
  }

  /**
   * Commits a modified log CEL filter expression text queries.
   */
  protected onLogCelFilterChange(filter: string): void {
    this.logCelFilter$.next(filter);
  }

  /**
   * Triggers the rendering graph outputs.
   */
  protected onDrawDiagram() {
    const targetPath = window.location.pathname.endsWith('/')
      ? window.location.pathname + 'graph'
      : window.location.pathname + '/graph';
    window.open(targetPath, '_blank');
  }

  /**
   * Updates the state visibility for timelines missing log hits.
   */
  protected onToggleHideTimelinesWithoutMatchingLogs(value: boolean): void {
    this.viewStateService.setHideTimelinesWithoutMatchingLogs(value);
  }
}

/**
 * Compiles log search query and severity into a CEL expression.
 */
export function compileLogFiltersToCel(
  severity: string,
  searchQuery: string,
): string {
  const parts: string[] = [];
  if (severity && severity !== 'ANY') {
    parts.push(`severity >= ${severity}`);
  }
  if (searchQuery && searchQuery.trim() !== '') {
    const escaped = searchQuery.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
    parts.push(`body("${escaped}")`);
  }
  return parts.join(' && ');
}

/**
 * Compiles a list of standard timeline filters into a CEL expression.
 */
export function compileFiltersToCel(filters: TimelineFilterConfig[]): string {
  if (filters.length === 0) {
    return '';
  }
  return filters
    .map((f) => {
      let celValue = f.value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
      if (f.mode === 'selection') {
        const escapedParts = f.value.split('|').map((val) =>
          val
            .replace(/\\/g, '\\\\')
            .replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
            .replace(/"/g, '\\"'),
        );
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
