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

import { CommonModule } from '@angular/common';
import { Component, computed, input, model, signal } from '@angular/core';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { OverlayModule, ConnectionPositionPair } from '@angular/cdk/overlay';
import {
  RevisionStateStyle,
  LogType,
  RevisionState,
} from 'src/app/store/domain/style';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { RendererConvertUtil } from './canvas/convertutil';
import { MarkdownPopupComponent } from './markdown-popup.component';

/**
 * ViewModel for revision legend item.
 */
interface RevisionLegendViewModel {
  label: string;
  icon: string;
  style: RevisionStateStyle;
  color: string;
  description: string;
}

/**
 * ViewModel for event legend item.
 */
interface EventLegendViewModel {
  label: string;
  color: string;
}

/**
 * ViewModel for timeline type legend item.
 */
interface TimelineTypeLegendViewModel {
  label: string;
  backgroundColor: string;
  color: string;
  hint: string;
}

/**
 * Component for displaying the legend of the timeline.
 * It shows the explanation of icons and colors used in the timeline.
 */
@Component({
  selector: 'khi-timeline-legend',
  templateUrl: './timeline-legend.component.html',
  styleUrls: ['./timeline-legend.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    MatButtonToggleModule,
    MatExpansionModule,
    OverlayModule,
    MarkdownPopupComponent,
  ],
})
export class TimelineLegendComponent {
  protected readonly RevisionStateStyle = RevisionStateStyle;

  /**
   * Whether the legend is expanded.
   */
  expanded = model(false);

  /**
   * The currently selected legend type ('revisions' or 'events').
   */
  legendType = model<string>('revisions');

  /**
   * The timeline data to generate legends for.
   */
  timeline = input<ReadonlyDomainElement<Timeline> | null>(null);

  /**
   * The label of the revision legend whose popup is currently open.
   */
  readonly activePopupLabel = signal<string | null>(null);

  /**
   * Delay in milliseconds before closing the popup to allow transition to the popup card.
   */
  private static readonly POPUP_CLOSE_DELAY_MS = 100;

  private _closeTimeout: ReturnType<typeof setTimeout> | null = null;

  /**
   * Connection position pairs for the description overlay.
   */
  readonly popupPositions: ConnectionPositionPair[] = [
    new ConnectionPositionPair(
      { originX: 'center', originY: 'top' },
      { overlayX: 'center', overlayY: 'bottom' },
      0,
      -8,
    ),
    new ConnectionPositionPair(
      { originX: 'center', originY: 'bottom' },
      { overlayX: 'center', overlayY: 'top' },
      0,
      8,
    ),
  ];

  /**
   * Opens the description popup for the specified legend.
   */
  openPopup(legend: RevisionLegendViewModel): void {
    // Cancels any pending close timeout to keep the popup open when moving the cursor into the popup card.
    if (this._closeTimeout) {
      clearTimeout(this._closeTimeout);
      this._closeTimeout = null;
    }
    if (legend.description) {
      this.activePopupLabel.set(legend.label);
    }
  }

  /**
   * Closes the description popup.
   */
  closePopup(): void {
    // Delays closing to allow the cursor to transition from the trigger icon into the popup card without disappearing.
    this._closeTimeout = setTimeout(() => {
      this.activePopupLabel.set(null);
      this._closeTimeout = null;
    }, TimelineLegendComponent.POPUP_CLOSE_DELAY_MS);
  }

  /**
   * Computed ViewModel for the timeline type legend.
   */
  timelineTypeLegend = computed<TimelineTypeLegendViewModel | null>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return null;
    }

    return {
      label: timeline.type.label,
      color: RendererConvertUtil.hdrColorToCSSColor([
        timeline.type.foregroundColor.r,
        timeline.type.foregroundColor.g,
        timeline.type.foregroundColor.b,
        timeline.type.foregroundColor.a,
      ]),
      backgroundColor: RendererConvertUtil.hdrColorToCSSColor([
        timeline.type.backgroundColor.r,
        timeline.type.backgroundColor.g,
        timeline.type.backgroundColor.b,
        timeline.type.backgroundColor.a,
      ]),
      hint: timeline.type.description,
    };
  });

  /**
   * Computed list of ViewModels for revision legends found in the timeline.
   */
  revisionLegends = computed<RevisionLegendViewModel[]>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return [];
    }
    const revisionStateIds = new Set<number>();
    const uniqueStates: RevisionState[] = [];
    for (const revision of timeline.revisions) {
      const state = revision.state;
      if (!revisionStateIds.has(state.id)) {
        revisionStateIds.add(state.id);
        uniqueStates.push(state);
      }
    }
    return uniqueStates.map<RevisionLegendViewModel>((state) => {
      return {
        label: state.label,
        icon: state.icon,
        style: state.style,
        color: RendererConvertUtil.hdrColorToCSSColor([
          state.backgroundColor.r,
          state.backgroundColor.g,
          state.backgroundColor.b,
          state.backgroundColor.a,
        ]),
        description: state.description,
      };
    });
  });

  /**
   * Computed list of ViewModels for event legends found in the timeline.
   */
  eventLegends = computed<EventLegendViewModel[]>(() => {
    const timeline = this.timeline();
    if (timeline === null) {
      return [];
    }
    const eventTypeIds = new Set<number>();
    const uniqueTypes: LogType[] = [];
    for (const event of timeline.events) {
      const type = event.log.logType;
      if (!eventTypeIds.has(type.id)) {
        eventTypeIds.add(type.id);
        uniqueTypes.push(type);
      }
    }
    return uniqueTypes.map<EventLegendViewModel>((type) => {
      return {
        label: type.label,
        color: RendererConvertUtil.hdrColorToCSSColor([
          type.backgroundColor.r,
          type.backgroundColor.g,
          type.backgroundColor.b,
          type.backgroundColor.a,
        ]),
      };
    });
  });
}
