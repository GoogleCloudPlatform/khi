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
  ViewChild,
  inject,
  model,
} from '@angular/core';
import {
  BehaviorSubject,
  Subject,
  combineLatest,
  combineLatestWith,
  filter,
  map,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
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
import {
  ResourceRevisionChangePair,
  ResourceTimeline,
  TimelineLayer,
} from '../store/timeline';
import { ResourceRevision } from '../store/revision';
import { CommonModule } from '@angular/common';
import { ParsePrincipalPipe } from './diff-view-pipes';
import { TimestampFormatPipe } from '../common/timestamp-format.pipe';
import { UnifiedDiffComponent } from 'ngx-diff';
import { HighlightModule } from 'ngx-highlightjs';
import { AngularSplitModule } from 'angular-split';
import { toObservable } from '@angular/core/rxjs-interop';
import * as yaml from 'js-yaml';
import { DiffToolbarComponent } from './components/diff-toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';

class DiffViewScrollStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(12, 100, 1000);
  }
}

interface DiffViewSelectionMoveCommand {
  direction: 'next' | 'prev';
}

type DiffViewViewModel = {
  selectedTimeline: ResourceTimeline | null;
  selectedLogIndex: number;
  highlightedLogIndex: Set<number>;
  currentRevision: ResourceRevision | null;
  previousRevision: ResourceRevision | null;
  currentRevisionContent: string;
  previousRevisionContent: string;
};

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

  private destoroyed = new Subject<void>();

  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  @ViewChild(CdkVirtualScrollViewport) viewPort!: CdkVirtualScrollViewport;

  public timeline = new BehaviorSubject<ResourceTimeline | null>(null);

  timelineAnnotators = this.timelineAnnotatorResolver.getResolvedAnnotators(
    this.timeline,
    this.envInjector,
  );

  public currentRevision = new BehaviorSubject<ResourceRevision | null>(null);

  public $previousRevision = this._selectionManager.previousOfSelectedRevision;

  changePair = combineLatest([
    this.$previousRevision,
    this.currentRevision,
  ]).pipe(
    filter(([, current]) => !!current),
    map(([prev, current]) => new ResourceRevisionChangePair(prev, current!)),
  );

  changePairAnnotators = this.changePairAnnotatorResolver.getResolvedAnnotators(
    this.changePair,
    this.envInjector,
  );

  public $selectedLogIndex = this._selectionManager.selectedLogIndex;

  public $highlightLogIndex = this._selectionManager.highlightLogIndices;

  protected readonly showManagedFields = model(false);

  public diffViewViewModel = this._selectionManager.selectedRevision.pipe(
    combineLatestWith(
      this._selectionManager.previousOfSelectedRevision,
      this._selectionManager.selectedTimeline,
      this._selectionManager.selectedLog,
      this._selectionManager.highlightedLogs,
      toObservable(this.showManagedFields),
    ),
    map(([c, r, timeline, selectedLog, highlightedLogs, showManagedFields]) => {
      const currentContent = c?.resourceContent ?? '';
      const previousContent = r?.resourceContent ?? '';
      return {
        currentRevision: c,
        previousRevision: r,
        selectedTimeline: timeline,
        selectedLogIndex: selectedLog?.logIndex ?? -1,
        highlightedLogIndex: new Set(highlightedLogs.map((l) => l.logIndex)),
        currentRevisionContent: showManagedFields
          ? currentContent
          : this.removeManagedField(currentContent),
        previousRevisionContent: showManagedFields
          ? previousContent
          : this.removeManagedField(previousContent),
      } as DiffViewViewModel;
    }),
  );

  public $logs = this._inspectionDataStore.allLogs;

  diffViewSelectionMoveCommand = new Subject<DiffViewSelectionMoveCommand>();

  disableScrollForNext = false;

  ngOnInit(): void {
    this._initBindingLogSelectEvent();
    this._selectionManager.selectedTimeline
      .pipe(takeUntil(this.destoroyed))
      .subscribe(this.timeline);
    this._selectionManager.selectedRevision
      .pipe(takeUntil(this.destoroyed))
      .subscribe(this.currentRevision);
    this.diffViewSelectionMoveCommand
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(this.currentRevision, this.timeline),
      )
      .subscribe(([command, revision, timeline]) => {
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
    this._selectionManager.changeSelectionByRevision(this.timeline.value!, r);
  }

  _highlightRevision(r: ResourceRevision) {
    this._selectionManager.onHighlightLog(r.logIndex);
  }

  private _initBindingLogSelectEvent() {
    this.$selectedLogIndex
      .pipe(takeUntil(this.destoroyed), withLatestFrom(this.timeline))
      .subscribe(([index, timeline]) => {
        // Ignore selection event fired from a selection event on diff view
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
            this.viewPort?.scrollToIndex(revisionIndex, 'smooth');
          }
        }
      });
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
    const currentTimeline = this.timeline.value;
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
        `/diff/${kind}/${namespace}/${name}/${subresource}?logIndex=${this.currentRevision.value?.logIndex}`,
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
