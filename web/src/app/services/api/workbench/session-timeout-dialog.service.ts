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

import { Injectable, effect, inject, untracked } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import {
  openSessionTimedOutDialog,
  SessionTimedOutSmartComponent,
} from 'src/app/dialogs/session-timed-out/session-timed-out-smart.component';

/**
 * Monitors session timeout status and manages the lifecycle of the timeout dialog.
 */
@Injectable({
  providedIn: 'root',
})
export class SessionTimeoutDialogService {
  private readonly dialog = inject(MatDialog);
  private readonly workbenchClient = inject(WorkbenchClientService);

  private activeDialogRef: MatDialogRef<
    SessionTimedOutSmartComponent,
    void
  > | null = null;

  constructor() {
    effect(() => {
      const isExpired = this.workbenchClient.isWorkbenchExpired();

      if (isExpired) {
        untracked(() => {
          if (!this.activeDialogRef) {
            this.openDialog();
          }
        });
      } else {
        untracked(() => {
          if (this.activeDialogRef) {
            this.activeDialogRef.close();
            this.activeDialogRef = null;
          }
        });
      }
    });
  }

  /**
   * Opens the Session Timed Out dialog if not already open.
   */
  private openDialog(): void {
    if (this.activeDialogRef) {
      return;
    }

    const dialogRef = openSessionTimedOutDialog(this.dialog);
    this.activeDialogRef = dialogRef;
    dialogRef.afterClosed().subscribe(() => {
      if (this.activeDialogRef === dialogRef) {
        this.activeDialogRef = null;
      }
    });
  }
}
