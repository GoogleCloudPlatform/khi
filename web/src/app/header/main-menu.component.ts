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

import { Component } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { StartupDialogComponent } from '../dialogs/startup/startup.component';

@Component({
  selector: 'khi-main-menu',
  templateUrl: './main-menu.component.html',
  styleUrls: ['./main-menu.component.sass'],
})
export class MainMenuComponent {
  constructor(private readonly dialog: MatDialog) {}

  openStartupMenu() {
    this.dialog.open(StartupDialogComponent, {
      maxWidth: '100vw',
      panelClass: 'startup-modalbox',
    });
  }
}
