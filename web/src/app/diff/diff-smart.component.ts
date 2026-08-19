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
  OnInit,
  computed,
  inject,
  model,
  resource,
} from '@angular/core';
import { Subject, takeUntil } from 'rxjs';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import {
  SearchScope,
  ViewStateService,
} from 'src/app/services/view-state.service';
import { DiffListHeaderComponent } from './components/diff-list-header.component';
import { DiffListComponent } from './components/diff-list.component';
import { DiffContentComponent } from './components/diff-content.component';
import { CommonModule } from '@angular/common';
import { AngularSplitModule } from 'angular-split';
import { toSignal } from '@angular/core/rxjs-interop';
import * as yaml from 'js-yaml';

import { Revision } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

interface DiffSmartSelectionMoveCommand {
  direction: 'next' | 'prev';
}

/**
 * Component for displaying the difference between two selected resource revisions.
 * Acts as a smart container delegating presentation to header, list, and content components.
 */
@Component({
  selector: 'khi-diff-smart',
  templateUrl: './diff-smart.component.html',
  styleUrls: ['./diff-smart.component.scss'],
  imports: [
    CommonModule,
    AngularSplitModule,
    DiffListHeaderComponent,
    DiffListComponent,
    DiffContentComponent,
  ],
})
export class DiffSmartComponent implements OnInit, OnDestroy {
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly selectionManager = inject(SelectionManager);
  private readonly viewState = inject(ViewStateService);
  private readonly workbenchClientService = inject(WorkbenchClientService);
  private readonly currentRevisionResource = resource({
    params: () => this.selectionManager.selectedRevision()?.structId,
    loader: async ({ params: structId }) => {
      if (!structId || structId <= 0) {
        return '';
      }
      try {
        return await this.workbenchClientService.readStructYAML(structId);
      } catch (err) {
        console.warn(
          `[DiffSmartComponent] Failed to read struct YAML for structId ${structId}:`,
          err,
        );
        return '';
      }
    },
  });
  private readonly previousRevisionResource = resource({
    params: () => this.selectionManager.previousOfSelectedRevision()?.structId,
    loader: async ({ params: structId }) => {
      if (!structId || structId <= 0) {
        return null;
      }
      try {
        return await this.workbenchClientService.readStructYAML(structId);
      } catch (err) {
        console.warn(
          `[DiffSmartComponent] Failed to read struct YAML for previous structId ${structId}:`,
          err,
        );
        return null;
      }
    },
  });
  private destroyed = new Subject<void>();

  ngOnDestroy(): void {
    this.destroyed.next();
  }

  /**
   * Signal indicating whether either current or previous revision YAML is currently being loaded.
   */
  public readonly isLoading = computed(
    () =>
      this.currentRevisionResource.isLoading() ||
      this.previousRevisionResource.isLoading(),
  );

  /** Holds the active search scope. */
  public readonly activeSearchScope = this.viewState.activeSearchScope;

  /**
   * Signal containing the timezone shift in hours from the view state.
   */
  public readonly timezoneShift = toSignal(this.viewState.timezoneShift, {
    initialValue: 0,
  });

  /**
   * Signal containing the locally selected log index managed by SelectionManager.
   */
  protected readonly selectedLogIndex = this.selectionManager.selectedLogIndex;

  /**
   * Signal containing the set of highlighted log indices.
   */
  protected readonly highlightedLogIndices =
    this.selectionManager.highlightLogIndices;

  /**
   * Signal containing the timeline of the currently selected revision/log, or the selected timeline if none is selected.
   */
  protected readonly selectedTimeline = computed(() => {
    const revision = this.currentRevision();
    if (revision) {
      return revision.timeline;
    }
    const log = this.selectionManager.selectedLog();
    if (log) {
      for (const t of this.selectionManager.selectedTimelinesWithChildren()) {
        if (t.lookupRevisionFromLog(log) || t.lookupEventFromLog(log)) {
          return t;
        }
      }
    }
    return this.selectionManager.selectedTimeline();
  });

  /**
   * Signal containing the currently selected resource revision.
   */
  protected readonly currentRevision = this.selectionManager.selectedRevision;

  /**
   * Computed raw string of the current revision's content before stripping managed fields.
   */
  protected readonly currentRevisionRawBody = computed(() => {
    return this.currentRevisionResource.value() ?? '';
  });

  /**
   * Computed string of the current revision's content, formatted according to managed fields visibility.
   */
  protected readonly currentRevisionContent = computed(() => {
    const content = this.currentRevisionResource.value() ?? '';
    return this.showManagedFields()
      ? content
      : this.removeManagedField(content);
  });

  /**
   * Signal containing the revision immediately preceding the currently selected one.
   */
  protected readonly previousRevision =
    this.selectionManager.previousOfSelectedRevision;

  /**
   * Computed string of the previous revision's content, formatted according to managed fields visibility.
   */
  protected readonly previousRevisionContent = computed<string | null>(() => {
    const content = this.previousRevisionResource.value();
    if (content === undefined || content === null) {
      return null;
    }
    return this.showManagedFields()
      ? content
      : this.removeManagedField(content);
  });

  /**
   * Model to toggle the visibility of Kubernetes managed fields in the diff view.
   */
  protected readonly showManagedFields = model(false);

  /**
   * Signal containing all log entries available in the inspection data store.
   */
  public readonly allLogs = computed(() => {
    const data = this.inspectionDataStore.inspectionData();
    return data ? Array.from(data.logStore.logs()) : [];
  });

  /**
   * Subject to propagate keyboard selection commands (up/down).
   */
  diffSmartSelectionMoveCommand = new Subject<DiffSmartSelectionMoveCommand>();

  ngOnInit(): void {
    this.diffSmartSelectionMoveCommand
      .pipe(takeUntil(this.destroyed))
      .subscribe((command) => {
        const revision = this.currentRevision();
        const timeline = this.selectedTimeline();
        if (revision === null || timeline === null) return;
        const direction = command.direction === 'prev' ? -1 : 1;
        const revIndex = timeline.revisions.indexOf(revision);
        if (revIndex === -1) return;
        const nextSelected = Math.max(
          0,
          Math.min(timeline.revisions.length - 1, revIndex + direction),
        );
        const next = timeline.revisions[nextSelected];
        if (next.logIndex !== -1) {
          this.selectionManager.onSelectRevision(next);
        }
      });
  }

  /**
   * Handles explicitly selecting a revision from the list.
   * @param r The resource revision clicked by the user.
   */
  _selectRevision(r: ReadonlyDomainElement<Revision>) {
    this.selectionManager.onSelectRevision(r);
  }

  /**
   * Triggers highlighting for a specific log index corresponding to the hovered revision.
   * @param r The resource revision hovered by the user.
   */
  _highlightRevision(r: ReadonlyDomainElement<Revision>) {
    this.selectionManager.onHighlightLog(r.log);
  }

  /**
   * Emits a sequence command (arrow up/down) to adjust the selected revision.
   * @param direction 'next' for down-arrow, 'prev' for up-arrow
   */
  onMoveSelection(direction: 'next' | 'prev') {
    this.diffSmartSelectionMoveCommand.next({ direction });
  }

  /**
   * Opens the current diff view in a separate window tab.
   */
  openDiffInAnotherWindow() {
    const currentTimeline = this.selectedTimeline();
    if (!currentTimeline) {
      return;
    }
    window.open(
      window.location.pathname +
        `/diff?timeline=${currentTimeline.id}&logIndex=${this.currentRevision()?.logIndex}`,
      '_blank',
    );
  }

  /**
   * Utility to safely remove Kubernetes managed fields from a YAML text resource representation.
   * @param content The original YAML string.
   * @returns Cleaned text string without managedFields, or the original on error.
   */
  private removeManagedField(content: string): string {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const yamlData = yaml.load(content) as any;
      if (
        yamlData &&
        yamlData['metadata'] &&
        yamlData['metadata']['managedFields']
      ) {
        delete yamlData.metadata.managedFields;
      }
      return yamlData ? yaml.dump(yamlData, { lineWidth: -1 }) : content;
    } catch (e) {
      console.warn(`failed to process frontend yaml: ${e}`);
      return content;
    }
  }

  /**
   * Sets the active search scope in the ViewStateService based on whether Diff Content is hovered or focused.
   */
  protected onScopeActiveChange(active: boolean): void {
    if (active) {
      this.viewState.activeSearchScope.set(SearchScope.Diff);
    } else if (this.viewState.activeSearchScope() === SearchScope.Diff) {
      this.viewState.activeSearchScope.set(SearchScope.Global);
    }
  }
}
