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
} from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { BreakpointObserver } from '@angular/cdk/layout';
import {
  Subject,
  debounceTime,
  distinctUntilChanged,
  from,
  map,
  of,
  switchMap,
  takeUntil,
} from 'rxjs';
import { ViewStateService } from 'src/app/services/view-state.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { CelValidationClientService } from 'src/app/services/api/cel/cel-validation-client.service';
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
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly celValidationClient = inject(CelValidationClientService);
  private readonly selectionManager = inject(SelectionManager);
  private readonly breakpointObserver = inject(BreakpointObserver);

  private readonly destroyed = new Subject<void>();

  // Global state
  /** Global display mode state signal managed by ViewStateService. */
  protected readonly isAdvancedMode = this.viewStateService.isAdvancedMode;

  /** Signal holding the current active search scope. */
  protected readonly activeSearchScope =
    this.viewStateService.activeSearchScope;

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
      .filter((t: TimelineType) => activeLabels.has(t.label.toLowerCase()))
      .sort((a: TimelineType, b: TimelineType) =>
        a.label.localeCompare(b.label),
      );
  });

  /**
   * Returns a list of label suggestions for a given timeline type.
   */
  private getCandidatesForType(typeLabel: string): string[] {
    const store = this.inspectionData()?.timelineStore;
    if (!store) {
      return [];
    }
    const candSet = new Set<string>();
    const typeLower = typeLabel.toLowerCase();
    for (const t of store.timelines) {
      for (const node of t.path) {
        if (node.type.label.toLowerCase() === typeLower) {
          candSet.add(node.label);
        }
      }
    }
    return Array.from(candSet).sort();
  }

  /** Interactive timeline label suggestions list. */
  protected readonly candidates = computed<string[]>(() => {
    const selectedType = this.selectedTimelineTypeForBuilder();
    if (selectedType === '*') {
      return [];
    }
    return this.getCandidatesForType(selectedType);
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
  /** Active advanced timeline include CEL expression signal. */
  protected readonly timelineIncludeCelFilter =
    this.viewStateService.advancedTimelineIncludeCel.asReadonly();

  /** Active advanced timeline exclude CEL expression signal. */
  protected readonly timelineExcludeCelFilter =
    this.viewStateService.advancedTimelineExcludeCel.asReadonly();

  /** Active advanced log CEL expression signal. */
  protected readonly logCelFilter =
    this.viewStateService.advancedLogCel.asReadonly();

  /** Validation result error output for timeline include CEL queries. */
  protected readonly timelineIncludeCelError = signal<string>('');

  /** Validation result error output for timeline exclude CEL queries. */
  protected readonly timelineExcludeCelError = signal<string>('');

  /** Validation result error output for log CEL queries. */
  protected readonly logCelError = signal<string>('');

  /** Active option toggle matching log hits. */
  protected readonly hideTimelinesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideTimelinesWithoutMatchingLogs,
    { initialValue: true },
  );

  constructor() {
    // Synchronize hideTimelinesWithoutMatchingLogs to backend filter
    effect(() => {
      const view = this.inspectionDataStore.timelineView();
      if (!view) return;

      const hideNoLogs = this.hideTimelinesWithoutMatchingLogs() ?? true;
      view.backendFilter.updateFilterParams({
        excludeNoLogs: hideNoLogs,
      });
    });

    // Synchronize standard filter changes to advanced CEL inputs and backend filter
    effect(() => {
      if (!this.isAdvancedMode()) {
        const filters = this.timelineFilters();
        const severity = this.selectedSeverity();
        const searchQuery = this.logSearchQuery();

        const timelineQuery = compileFiltersToCel(filters, severity);
        const timelineExclusionQuery = compileExclusionFiltersToCel(filters);
        const logQuery = compileLogFiltersToCel(severity, searchQuery);

        this.viewStateService.advancedTimelineIncludeCel.set(timelineQuery);
        this.viewStateService.advancedTimelineExcludeCel.set(
          timelineExclusionQuery,
        );
        this.viewStateService.advancedLogCel.set(logQuery);

        this.timelineIncludeCelError.set('');
        this.timelineExcludeCelError.set('');
        this.logCelError.set('');

        const view = this.inspectionDataStore.timelineView();
        if (view) {
          view.backendFilter.updateFilterParams({
            timelineQuery,
            timelineExclusionQuery,
            logQuery,
          });
        }
      }
    });

    // Debounced validation and application for Advanced Mode timeline include CEL queries
    toObservable(this.viewStateService.advancedTimelineIncludeCel)
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        switchMap((val) => {
          if (!this.isAdvancedMode()) {
            return of({ query: val, valid: true, error: '' });
          }
          if (!val || val.trim() === '') {
            return of({ query: '', valid: true, error: '' });
          }
          return from(this.celValidationClient.validateTimelineQuery(val)).pipe(
            map((res) => ({
              query: val,
              valid: res.valid,
              error: res.valid ? '' : res.errorMessage,
            })),
          );
        }),
        takeUntil(this.destroyed),
      )
      .subscribe((result) => {
        if (this.isAdvancedMode()) {
          this.timelineIncludeCelError.set(result.error);
        }
        if (result.valid && this.isAdvancedMode()) {
          const view = this.inspectionDataStore.timelineView();
          if (view) {
            view.backendFilter.updateFilterParams({
              timelineQuery: result.query,
            });
          }
        }
      });

    // Debounced validation and application for Advanced Mode timeline exclude CEL queries
    toObservable(this.viewStateService.advancedTimelineExcludeCel)
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        switchMap((val) => {
          if (!this.isAdvancedMode()) {
            return of({ query: val, valid: true, error: '' });
          }
          if (!val || val.trim() === '') {
            return of({ query: '', valid: true, error: '' });
          }
          return from(this.celValidationClient.validateTimelineQuery(val)).pipe(
            map((res) => ({
              query: val,
              valid: res.valid,
              error: res.valid ? '' : res.errorMessage,
            })),
          );
        }),
        takeUntil(this.destroyed),
      )
      .subscribe((result) => {
        if (this.isAdvancedMode()) {
          this.timelineExcludeCelError.set(result.error);
        }
        if (result.valid && this.isAdvancedMode()) {
          const view = this.inspectionDataStore.timelineView();
          if (view) {
            view.backendFilter.updateFilterParams({
              timelineExclusionQuery: result.query,
            });
          }
        }
      });

    // Debounced validation and application for Advanced Mode log CEL queries
    toObservable(this.viewStateService.advancedLogCel)
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        switchMap((val) => {
          if (!this.isAdvancedMode()) {
            return of({ query: val, valid: true, error: '' });
          }
          if (!val || val.trim() === '') {
            return of({ query: '', valid: true, error: '' });
          }
          return from(this.celValidationClient.validateLogQuery(val)).pipe(
            map((res) => ({
              query: val,
              valid: res.valid,
              error: res.valid ? '' : res.errorMessage,
            })),
          );
        }),
        takeUntil(this.destroyed),
      )
      .subscribe((result) => {
        if (this.isAdvancedMode()) {
          this.logCelError.set(result.error);
        }
        if (result.valid && this.isAdvancedMode()) {
          const view = this.inspectionDataStore.timelineView();
          if (view) {
            view.backendFilter.updateFilterParams({
              logQuery: result.query,
            });
          }
        }
      });
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
   * Commits a modified timeline include CEL filter expression text queries.
   */
  protected onTimelineIncludeCelFilterChange(filter: string): void {
    this.viewStateService.advancedTimelineIncludeCel.set(filter);
  }

  /**
   * Commits a modified timeline exclude CEL filter expression text queries.
   */
  protected onTimelineExcludeCelFilterChange(filter: string): void {
    this.viewStateService.advancedTimelineExcludeCel.set(filter);
  }

  /**
   * Commits a modified log CEL filter expression text queries.
   */
  protected onLogCelFilterChange(filter: string): void {
    this.viewStateService.advancedLogCel.set(filter);
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
 * Compiles a list of standard timeline filters and a severity level into a CEL expression for inclusion.
 */
export function compileFiltersToCel(
  filters: TimelineFilterConfig[],
  severity: string = 'ANY',
): string {
  const parts: string[] = [];
  if (severity && severity !== 'ANY') {
    parts.push(`minSeverity(${severity})`);
  }
  const includeFilters = filters.filter((f) => f.action !== 'exclude');
  if (includeFilters.length > 0) {
    const filtersCel = includeFilters
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
    parts.push(filtersCel);
  }
  return parts.join(' && ');
}

/**
 * Compiles a list of standard timeline exclusion filters into a CEL expression for exclusion.
 */
export function compileExclusionFiltersToCel(
  filters: TimelineFilterConfig[],
): string {
  const excludeFilters = filters.filter((f) => f.action === 'exclude');
  if (excludeFilters.length === 0) {
    return '';
  }
  return excludeFilters
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
    .join(' || ');
}
