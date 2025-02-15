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

import { Inject, Injectable } from '@angular/core';
import { SelectionManagerService } from '../services/selection-manager.service';
import { Subject, withLatestFrom } from 'rxjs';
import { ResourceTimeline, TimelineLayer } from '../store/timeline';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';

interface MoveHorizontalSelectionCommand {
  direction: 'prev' | 'next';
  toEnd: boolean;
}

type MoveVerticalSelectionCommandMode =
  | 'default'
  | 'resource'
  | 'namespace'
  | 'kind';

interface MoveVerticalSelectionCommand {
  direction: 'prev' | 'next';
  mode: MoveVerticalSelectionCommandMode;
}

/**
 * CanvasKeyEventHandler handles keyboard event from canvas.
 */
@Injectable({ providedIn: 'root' })
export class CanvasKeyEventHandler {
  private moveHorizontalSelectionCommand =
    new Subject<MoveHorizontalSelectionCommand>();

  private moveVerticalSelectionCommand =
    new Subject<MoveVerticalSelectionCommand>();

  constructor(
    private selectionManager: SelectionManagerService,
    @Inject(DEFAULT_TIMELINE_FILTER) filter: TimelineFilter,
  ) {
    // For revision selection
    this.moveHorizontalSelectionCommand
      .pipe(
        withLatestFrom(
          selectionManager.selectedRevision,
          selectionManager.selectedTimeline,
        ),
      )
      .subscribe(([command, revision, timeline]) => {
        if (timeline === null || revision === null) return;
        const revIndex = timeline.revisions.indexOf(revision);
        if (revIndex === -1) return;
        // Some timelines can have pseudo revision at the first.
        const minimumIndex =
          timeline.revisions.length > 0 && timeline.revisions[0].logIndex == -1
            ? 1
            : 0;
        const direction = command.direction === 'prev' ? -1 : 1;
        const diff = command.toEnd
          ? direction * timeline.revisions.length
          : direction;
        const nextIndex = Math.max(
          minimumIndex,
          Math.min(timeline.revisions.length - 1, revIndex + diff),
        );
        const next = timeline.revisions[nextIndex];
        this.selectionManager.changeSelectionByRevision(timeline, next);
      });
    // For event selection
    this.moveHorizontalSelectionCommand
      .pipe(
        withLatestFrom(
          selectionManager.selectedTimeline,
          selectionManager.selectedLogIndex,
        ),
      )
      .subscribe(([command, timeline, logIndex]) => {
        if (timeline === null) return;
        const currentEvent = timeline.events.find(
          (ev) => ev.logIndex === logIndex,
        );

        if (!currentEvent) return; // the log would be revision not event
        const currentEventIndex = timeline.events.indexOf(currentEvent);
        const direction = command.direction === 'prev' ? -1 : 1;
        const diff = command.toEnd
          ? direction * timeline.events.length
          : direction;
        const nextIndex = Math.max(
          0,
          Math.min(timeline.events.length - 1, currentEventIndex + diff),
        );
        const next = timeline.events[nextIndex];
        this.selectionManager.changeSelectionByEvent(timeline, next);
      });

    // vertical move
    this.moveVerticalSelectionCommand
      .pipe(
        withLatestFrom(
          filter.filteredTimeline,
          selectionManager.selectedTimeline,
        ),
      )
      .subscribe(([command, timelines, selectedTimeline]) => {
        if (selectedTimeline === null) return;
        const direction = command.direction === 'prev' ? -1 : 1;
        const timelineIndex = timelines.indexOf(selectedTimeline);
        let nextTimelineIndex = timelineIndex;
        switch (command.mode) {
          case 'default':
            nextTimelineIndex = Math.max(
              0,
              Math.min(timelines.length - 1, timelineIndex + direction),
            );
            break;
          case 'resource':
            nextTimelineIndex = this.findNextTimeline(
              timelines,
              timelineIndex,
              direction,
              TimelineLayer.Name,
            );
            break;
          case 'kind':
            nextTimelineIndex = this.findNextTimeline(
              timelines,
              timelineIndex,
              direction,
              TimelineLayer.Kind,
            );
            break;
          case 'namespace':
            nextTimelineIndex = this.findNextTimeline(
              timelines,
              timelineIndex,
              direction,
              TimelineLayer.Namespace,
            );
            break;
        }
        if (nextTimelineIndex !== -1)
          this.selectionManager.onSelectTimeline(timelines[nextTimelineIndex]);
      });
  }

  public keydown(keyEvent: KeyboardEvent): void {
    if (keyEvent.key === 'ArrowLeft') {
      this.moveHorizontalSelectionCommand.next({
        direction: 'prev',
        toEnd: keyEvent.altKey,
      });
      return;
    }
    if (keyEvent.key === 'ArrowRight') {
      this.moveHorizontalSelectionCommand.next({
        direction: 'next',
        toEnd: keyEvent.altKey,
      });
      return;
    }
    if (keyEvent.key === 'ArrowDown') {
      this.moveVerticalSelectionCommand.next({
        direction: 'next',
        mode: this.toVerticalMoveMode(keyEvent),
      });
      keyEvent.preventDefault();
      return;
    }
    if (keyEvent.key === 'ArrowUp') {
      this.moveVerticalSelectionCommand.next({
        direction: 'prev',
        mode: this.toVerticalMoveMode(keyEvent),
      });
      keyEvent.preventDefault();
      return;
    }
  }

  private toVerticalMoveMode(
    keyboardEvent: KeyboardEvent,
  ): MoveVerticalSelectionCommandMode {
    switch (true) {
      case keyboardEvent.altKey && keyboardEvent.shiftKey:
        return 'kind';
      case keyboardEvent.altKey && !keyboardEvent.shiftKey:
        return 'namespace';
      case !keyboardEvent.altKey && keyboardEvent.shiftKey:
        return 'resource';
      default:
        return 'default';
    }
  }

  private findNextTimeline(
    timelines: ResourceTimeline[],
    fromIndex: number,
    direction: number,
    layer: TimelineLayer,
  ): number {
    for (
      let i = fromIndex + direction;
      i >= 0 && i < timelines.length;
      i += direction
    ) {
      if (timelines[i].layer == layer) {
        return i;
      }
    }
    return -1;
  }
}
