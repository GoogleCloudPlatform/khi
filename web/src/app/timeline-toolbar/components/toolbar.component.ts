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
  HostListener,
  input,
  model,
  output,
  inject,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { OverlayModule } from '@angular/cdk/overlay';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatSnackBar } from '@angular/material/snack-bar';
import { RegexInputComponent } from './regex-input.component';
import { SetInputPopupComponent } from './set-input-popup.component';

export enum ToolbarPopupStatus {
  None = 'NONE_OPEN',
  KindFilter = 'KIND_FILTER_OPEN',
  NamespaceFilter = 'NAMESPACE_FILTER_OPEN',
  SubresourceFilter = 'SUBRESOURCE_FILTER_OPEN',
}

@Component({
  selector: 'khi-timeline-toolbar',
  templateUrl: './toolbar.component.html',
  styleUrls: ['./toolbar.component.scss'],
  imports: [
    CommonModule,
    SetInputPopupComponent,
    MatIconModule,
    OverlayModule,
    RegexInputComponent,
    MatButtonModule,
    MatButtonToggleModule,
  ],
})
export class ToolbarComponent {
  private readonly snackbar = inject(MatSnackBar);

  // Inputs (Signals)
  readonly showButtonLabel = input(false);
  readonly kinds = input<Set<string>>(new Set());
  readonly includedKinds = model<Set<string>>(new Set());
  readonly namespaces = input<Set<string>>(new Set());
  readonly includedNamespaces = model<Set<string>>(new Set());
  readonly subresourceRelationships = input<Set<string>>(new Set());
  readonly includedSubresourceRelationships = model<Set<string>>(new Set());
  readonly timezoneShift = model(0);
  readonly logOrTimelineNotSelected = input(true);
  readonly hideSubresourcesWithoutMatchingLogs = model(false);
  readonly hideResourcesWithoutMatchingLogs = model(false);
  readonly nameFilter = model('');
  readonly logFilter = model('');

  // Outputs (Outputs)
  readonly drawDiagram = output<void>();

  protected readonly ToolbarPopupStatus = ToolbarPopupStatus;

  protected popupStatus: ToolbarPopupStatus = ToolbarPopupStatus.None;

  protected setPopupState(state: ToolbarPopupStatus) {
    this.popupStatus =
      state === this.popupStatus ? ToolbarPopupStatus.None : state;
  }

  onTimezoneshiftCommit(event: Event) {
    const value = +(event.target as HTMLInputElement).value;
    this.timezoneShift.set(value);
  }

  @HostListener('window:keydown', ['$event'])
  protected interceptBrowserSearch(event: KeyboardEvent) {
    if (event.key === 'f' && (event.ctrlKey || event.metaKey)) {
      this.snackbar.open(
        'In-browser search may not work on KHI because elements outside the visible area are not rendered. Please use the search text field on the toolbar instead.',
        'OK',
      );
    }
  }
}
