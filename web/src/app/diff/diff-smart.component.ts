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
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import { ViewStateService } from '../services/view-state.service';
import { DiffListHeaderComponent } from './components/diff-list-header.component';
import { DiffListComponent } from './components/diff-list.component';
import { DiffContentComponent } from './components/diff-content.component';
import { CommonModule } from '@angular/common';
import { AngularSplitModule } from 'angular-split';
import { toSignal } from '@angular/core/rxjs-interop';
import { LogEntry } from '../store/log';
import * as yaml from 'js-yaml';
import { ResourceRevision } from '../store/revision';
import { ResourceRevisionChangePair, TimelineLayer } from '../store/timeline';

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
  private readonly _inspectionDataStore = inject(InspectionDataStoreService);
  private readonly _selectionManager = inject(SelectionManagerService);
  private readonly _viewState = inject(ViewStateService);
  private destoroyed = new Subject<void>();

  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  /**
   * Computed pair of the previous and current revision selected for diffing.
   */
  changePair = computed(() => {
    const prev = this.previousRevision();
    const current = this.currentRevision();
    return new ResourceRevisionChangePair(prev, current!);
  });

  /**
   * Signal containing the timezone shift in hours from the view state.
   */
  public readonly timezoneShift = toSignal(this._viewState.timezoneShift, {
    initialValue: 0,
  });

  /**
   * Signal containing the locally selected log index managed by SelectionManagerService.
   */
  protected readonly selectedLogIndex = toSignal(
    this._selectionManager.selectedLogIndex,
    { initialValue: -1 },
  );

  /**
   * Signal containing the set of highlighted log indices.
   */
  protected readonly highlightedLogIndices = toSignal(
    this._selectionManager.highlightLogIndices,
    { initialValue: new Set<number>() },
  );

  /**
   * Signal containing the currently selected resource timeline.
   */
  protected readonly selectedTimeline = toSignal(
    this._selectionManager.selectedTimeline,
    { initialValue: null },
  );

  /**
   * Signal containing the currently selected resource revision.
   */
  protected readonly currentRevision = toSignal(
    this._selectionManager.selectedRevision,
    { initialValue: null },
  );

  /**
   * Computed string of the current revision's content, formatted according to managed fields visibility.
   */
  protected readonly currentRevisionContent = computed(() => {
    const content = this.currentRevision()?.resourceContent ?? '';
    return this.showManagedFields()
      ? content
      : this.removeManagedField(content);
  });

  /**
   * Signal containing the revision immediately preceding the currently selected one.
   */
  protected readonly previousRevision = toSignal(
    this._selectionManager.previousOfSelectedRevision,
    { initialValue: null },
  );

  /**
   * Computed string of the previous revision's content, formatted according to managed fields visibility.
   */
  protected readonly previousRevisionContent = computed(() => {
    const content = this.previousRevision()?.resourceContent ?? '';
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
  public allLogs = toSignal(this._inspectionDataStore.allLogs, {
    initialValue: [] as LogEntry[],
  });

  /**
   * Subject to propagate keyboard selection commands (up/down).
   */
  diffSmartSelectionMoveCommand = new Subject<DiffSmartSelectionMoveCommand>();

  constructor() {}

  ngOnInit(): void {
    this.diffSmartSelectionMoveCommand
      .pipe(takeUntil(this.destoroyed))
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
          this._selectionManager.changeSelectionByRevision(timeline, next);
        }
      });
  }

  /**
   * Handles explicitly selecting a revision from the list.
   * @param r The resource revision clicked by the user.
   */
  _selectRevision(r: ResourceRevision) {
    this._selectionManager.changeSelectionByRevision(
      this.selectedTimeline()!,
      r,
    );
  }

  /**
   * Triggers highlighting for a specific log index corresponding to the hovered revision.
   * @param r The resource revision hovered by the user.
   */
  _highlightRevision(r: ResourceRevision) {
    this._selectionManager.onHighlightLog(r.logIndex);
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
    const kind = currentTimeline.getNameOfLayer(TimelineLayer.Kind);
    const namespace = currentTimeline.getNameOfLayer(TimelineLayer.Namespace);
    const name = currentTimeline.getNameOfLayer(TimelineLayer.Name);
    let subresource =
      currentTimeline.getNameOfLayer(TimelineLayer.Subresource) ?? '-';
    if (subresource == '') subresource = '-';
    window.open(
      window.location.pathname +
        `/diff/${kind}/${namespace}/${name}/${subresource}?logIndex=${this.currentRevision()?.logIndex}`,
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
}
