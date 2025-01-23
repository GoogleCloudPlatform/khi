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

import { Component, Inject } from '@angular/core';
import {
  PROGRESS_DIALOG_STATUS_OBSERVER,
  ProgressDialogStatusObserver,
} from 'src/app/services/progress/progress-interface';

@Component({
  templateUrl: './progress.component.html',
  styleUrls: ['./progress.component.sass'],
})
export class ProgressDialogComponent {
  public currentStatus = this.progressObserver.status();

  constructor(
    @Inject(PROGRESS_DIALOG_STATUS_OBSERVER)
    private progressObserver: ProgressDialogStatusObserver,
  ) {}
}
