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

import { Component, input, output } from '@angular/core';
import {
  InspectionTitleChangeRequest,
  TaskCardItemComponent,
  TaskCardItemViewModel,
} from './task-card-item.component';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'khi-task-card-list',
  imports: [
    MatProgressBarModule,
    MatIconModule,
    KHIIconRegistrationModule,
    TaskCardItemComponent,
  ],
  templateUrl: './task-card-list.component.html',
  styleUrls: ['./task-card-list.component.scss'],
})
export class TaskCardListComponent {
  tasks = input.required<TaskCardItemViewModel[] | undefined>();
  isViewerMode = input.required<boolean>();
  openInspectionResult = output<string>();
  openInspectionMetadata = output<string>();
  cancelInspection = output<string>();
  downloadInspectionResult = output<string>();
  changeInspectionTitle = output<InspectionTitleChangeRequest>();
}
