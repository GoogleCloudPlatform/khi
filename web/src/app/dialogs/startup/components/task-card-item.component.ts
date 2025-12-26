/**
 * Copyright 2025 Google LLC
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
  computed,
  effect,
  ElementRef,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatTooltipModule } from '@angular/material/tooltip';

export interface TaskCardItemViewModel {
  id: string;
  inspectionTimeLabel: string;
  label: string;
  phase: InspectionMetadataProgressPhase;
  totalProgress: TaskCardItemProgressBarViewModel;
  progresses: TaskCardItemProgressBarViewModel[];
  errors: TaskCardItemErrorViewModel[];
}

export interface TaskCardItemProgressBarViewModel {
  id: string;
  label: string;
  message: string;
  percentage: number;
  percentageLabel: string;
  indeterminate: boolean;
}

export interface TaskCardItemErrorViewModel {
  message: string;
  link: string;
}

export interface InspectionTitleChangeRequest {
  id: string;
  changeTo: string;
}

@Component({
  selector: 'khi-task-card-item',
  imports: [
    CommonModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './task-card-item.component.html',
  styleUrls: ['./task-card-item.component.scss'],
})
export class TaskCardItemComponent {
  task = input.required<TaskCardItemViewModel>();
  titleInput = viewChild<ElementRef<HTMLInputElement>>('titleInput');

  constructor() {
    effect(() => {
      if (this.isEditing()) {
        this.titleInput()?.nativeElement.focus();
      }
    });
  }

  openInspectionResult = output<string>();
  openInspectionMetadata = output<string>();
  cancelInspection = output<string>();
  downloadInspectionResult = output<string>();
  changeInspectionTitle = output<InspectionTitleChangeRequest>();

  isResultAvailable = computed(() => this.task().phase === 'DONE');
  isMetadataAvailbale = computed(
    () => this.task().phase === 'DONE' || this.task().phase === 'ERROR',
  );
  isCancellable = computed(() => this.task().phase === 'RUNNING');

  isEditing = signal(false);
  taskNameInput = signal('');

  startEditing() {
    this.isEditing.set(true);
    this.taskNameInput.set(this.task().label);
  }

  commitTitleChange() {
    this.isEditing.set(false);
    if (this.taskNameInput() !== this.task().label) {
      this.changeInspectionTitle.emit({
        id: this.task().id,
        changeTo: this.taskNameInput(),
      });
    }
  }

  cancelEditing() {
    this.isEditing.set(false);
  }
}
