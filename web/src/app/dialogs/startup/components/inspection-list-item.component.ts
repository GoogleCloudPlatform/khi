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

import {
  Component,
  input,
  output,
  computed,
  signal,
  viewChild,
  ElementRef,
  effect,
} from '@angular/core';
import { NgClass } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  InspectionListItemViewModel,
  InspectionTitleChangeRequest,
} from '../types/inspection-activity.model';

/**
 * Represents an item in the inspection list.
 * Displays progress, status, and commands for a single inspection task.
 */
@Component({
  selector: 'khi-inspection-list-item',
  imports: [
    NgClass,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './inspection-list-item.component.html',
  styleUrls: ['./inspection-list-item.component.scss'],
})
export class InspectionListItemComponent {
  /** The view model for the inspection item. */
  public readonly item = input.required<InspectionListItemViewModel>();

  /** Input element for editing the title. */
  private readonly titleInput =
    viewChild<ElementRef<HTMLInputElement>>('titleInput');

  constructor() {
    effect(() => {
      if (this.isEditing()) {
        const element = this.titleInput()?.nativeElement;
        if (element) {
          element.focus();
          element.select();
        }
      }
    });
  }

  /** Emitted when the user wants to open the inspection result. */
  public readonly openInspectionResult = output<string>();

  /** Emitted when the user wants to open the inspection metadata. */
  public readonly openInspectionMetadata = output<string>();

  /** Emitted when the user wants to cancel the running inspection. */
  public readonly cancelInspection = output<string>();

  /** Emitted when the user wants to download the inspection result. */
  public readonly downloadInspectionResult = output<string>();

  /** Emitted when the user changes the inspection title. */
  public readonly changeInspectionTitle =
    output<InspectionTitleChangeRequest>();

  /** Whether the inspection result is available for viewing. */
  protected readonly isResultAvailable = computed(
    () => this.item().phase === 'DONE',
  );

  /** Whether the inspection metadata is available for viewing. */
  protected readonly isMetadataAvailable = computed(
    () => this.item().phase === 'DONE' || this.item().phase === 'ERROR',
  );

  /** Whether the inspection can be cancelled. */
  protected readonly isCancellable = computed(
    () => this.item().phase === 'RUNNING',
  );

  /** Whether the title is currently being edited. */
  protected readonly isEditing = signal(false);

  /** Temporary input value for the task name during editing. */
  protected readonly taskNameInput = signal('');

  /** Enters editing mode. */
  protected startEditing(): void {
    this.isEditing.set(true);
    this.taskNameInput.set(this.item().label);
  }

  /** Commits the title change and exits editing mode. */
  protected commitTitleChange(): void {
    if (!this.isEditing()) {
      return;
    }
    this.isEditing.set(false);
    const newTitle = this.taskNameInput().trim();
    if (newTitle !== '' && newTitle !== this.item().label) {
      this.changeInspectionTitle.emit({
        id: this.item().id,
        changeTo: newTitle,
      });
    }
  }

  /** Cancels editing and reverts the title. */
  protected cancelEditing(): void {
    this.isEditing.set(false);
  }
}
