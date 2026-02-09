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
  EnvironmentInjector,
  OnDestroy,
  OnInit,
  computed,
  effect,
  inject,
  model,
  viewChild,
} from '@angular/core';
import { Subject, takeUntil } from 'rxjs';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import {
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  ScrollingModule,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import { TIMELINE_ANNOTATOR_RESOLVER } from '../annotator/timeline/resolver';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from '../annotator/change-pair/resolver';
import { ResourceRevisionChangePair, TimelineLayer } from '../store/timeline';
import { ResourceRevision } from '../store/revision';
import { CommonModule } from '@angular/common';
import { ParsePrincipalPipe } from './diff-view-pipes';
import { TimestampFormatPipe } from '../common/timestamp-format.pipe';
import { UnifiedDiffComponent } from 'ngx-diff';
import { HighlightModule } from 'ngx-highlightjs';
import { AngularSplitModule } from 'angular-split';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import * as yaml from 'js-yaml';
import { DiffToolbarComponent } from './components/diff-toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import { LogEntry } from '../store/log';

class DiffViewScrollStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(13, 100, 1000);
  }
}

interface DiffViewSelectionMoveCommand {
  direction: 'next' | 'prev';
}

@Component({
  selector: 'khi-diff-view',
  templateUrl: './diff-view.component.html',
  styleUrls: ['./diff-view.component.scss'],
  imports: [
    CommonModule,
    ScrollingModule,
    CdkVirtualScrollViewport,
    ParsePrincipalPipe,
    TimestampFormatPipe,
    UnifiedDiffComponent,
    HighlightModule,
    AngularSplitModule,
    DiffToolbarComponent,
  ],
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: DiffViewScrollStrategy },
  ],
})
export class DiffViewComponent implements OnInit, OnDestroy {
  private readonly _inspectionDataStore = inject(InspectionDataStoreService);
  private readonly _selectionManager = inject(SelectionManagerService);
  private readonly _clipboard = inject(Clipboard);
  private readonly _snackBar = inject(MatSnackBar);

  private readonly envInjector = inject(EnvironmentInjector);

  private readonly timelineAnnotatorResolver = inject(
    TIMELINE_ANNOTATOR_RESOLVER,
  );

  private readonly changePairAnnotatorResolver = inject(
    CHANGE_PAIR_ANNOTATOR_RESOLVER,
  );

  private readonly viewPort = viewChild(CdkVirtualScrollViewport);

  private destoroyed = new Subject<void>();

  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  changePair = computed(() => {
    const prev = this.previousRevision();
    const current = this.currentRevision();
    return new ResourceRevisionChangePair(prev, current!);
  });

  changePairAnnotators = this.changePairAnnotatorResolver.getResolvedAnnotators(
    toObservable(this.changePair),
    this.envInjector,
  );

  protected readonly selectedLogIndex = toSignal(
    this._selectionManager.selectedLogIndex,
  );

  protected readonly highlightedLogIndices = toSignal(
    this._selectionManager.highlightLogIndices,
    { initialValue: new Set<number>() },
  );

  protected readonly selectedTimeline = toSignal(
    this._selectionManager.selectedTimeline,
    { initialValue: null },
  );

  protected readonly currentRevision = toSignal(
    this._selectionManager.selectedRevision,
    { initialValue: null },
  );

  protected readonly currentRevisionContent = computed(() => {
    const content = this.currentRevision()?.resourceContent ?? '';
    return this.showManagedFields()
      ? content
      : this.removeManagedField(content);
  });

  protected readonly previousRevision = toSignal(
    this._selectionManager.previousOfSelectedRevision,
    { initialValue: null },
  );

  protected readonly previousRevisionContent = computed(() => {
    const content = this.previousRevision()?.resourceContent ?? '';
    return this.showManagedFields()
      ? content
      : this.removeManagedField(content);
  });

  protected readonly showManagedFields = model(false);

  timelineAnnotators = this.timelineAnnotatorResolver.getResolvedAnnotators(
    toObservable(this.selectedTimeline),
    this.envInjector,
  );

  public allLogs = toSignal(this._inspectionDataStore.allLogs, {
    initialValue: [] as LogEntry[],
  });

  diffViewSelectionMoveCommand = new Subject<DiffViewSelectionMoveCommand>();

  disableScrollForNext = false;

  constructor() {
    effect(() => {
      const index = this.selectedLogIndex();
      const timeline = this.selectedTimeline();
      const viewPort = this.viewPort();
      if (this.disableScrollForNext) {
        this.disableScrollForNext = false;
        return;
      }
      if (timeline === null) {
        return;
      }
      for (
        let revisionIndex = 0;
        revisionIndex < timeline.revisions.length;
        revisionIndex++
      ) {
        if (timeline.revisions[revisionIndex].logIndex === index) {
          viewPort?.scrollToIndex(revisionIndex, 'smooth');
        }
      }
    });
  }

  ngOnInit(): void {
    this.diffViewSelectionMoveCommand
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

  _selectRevision(r: ResourceRevision) {
    this.disableScrollForNext = true;
    this._selectionManager.changeSelectionByRevision(
      this.selectedTimeline()!,
      r,
    );
  }

  _highlightRevision(r: ResourceRevision) {
    this._selectionManager.onHighlightLog(r.logIndex);
  }

  public keyDown(keyEvent: KeyboardEvent) {
    if (keyEvent.key === 'ArrowDown') {
      this.diffViewSelectionMoveCommand.next({
        direction: 'next',
      });
      keyEvent.preventDefault();
    }
    if (keyEvent.key === 'ArrowUp') {
      this.diffViewSelectionMoveCommand.next({
        direction: 'prev',
      });
      keyEvent.preventDefault();
    }
  }

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

  copy(content: string) {
    let snackbarMessage = 'Copy failed';
    if (this._clipboard.copy(content)) {
      snackbarMessage = 'Copied!';
    }
    this._snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }

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
