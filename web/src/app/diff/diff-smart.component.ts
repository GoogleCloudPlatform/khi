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
} from '@angular/core';
import { Subject, takeUntil } from 'rxjs';
import { InspectionDataStoreV2 } from '../services/inspection-data-store-v2.service';
import { SelectionManagerV2 } from '../services/selection-manager-v2.service';
import { SearchScope, ViewStateService } from '../services/view-state.service';
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
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly viewState = inject(ViewStateService);
  private destroyed = new Subject<void>();

  ngOnDestroy(): void {
    this.destroyed.next();
  }

  /** Holds the active search scope. */
  public readonly activeSearchScope = this.viewState.activeSearchScope;

  /**
   * Signal containing the timezone shift in hours from the view state.
   */
  public readonly timezoneShift = toSignal(this.viewState.timezoneShift, {
    initialValue: 0,
  });

  /**
   * Signal containing the locally selected log index managed by SelectionManagerV2.
   */
  protected readonly selectedLogIndex = this.selectionManager.selectedLogIndex;

  /**
   * Signal containing the set of highlighted log indices.
   */
  protected readonly highlightedLogIndices =
    this.selectionManager.highlightLogIndices;

  /**
   * Signal containing the currently selected resource timeline.
   */
  protected readonly selectedTimeline = this.selectionManager.selectedTimeline;

  /**
   * Signal containing the currently selected resource revision.
   */
  protected readonly currentRevision = this.selectionManager.selectedRevision;

  /**
   * Computed string of the current revision's content, formatted according to managed fields visibility.
   */
  protected readonly currentRevisionContent = computed(() => {
    const content = this.currentRevision()?.bodyYAML ?? '';
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
  protected readonly previousRevisionContent = computed(() => {
    const content = this.previousRevision()?.bodyYAML ?? '';
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

  constructor() {}

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
