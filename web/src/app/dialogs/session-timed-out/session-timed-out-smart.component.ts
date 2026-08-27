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

import { Component, inject, signal } from '@angular/core';
import {
  MatDialog,
  MatDialogConfig,
  MatDialogRef,
} from '@angular/material/dialog';
import { SessionTimedOutLayoutComponent } from 'src/app/dialogs/session-timed-out/components/session-timed-out-layout.component';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import { openStartupDialog } from 'src/app/dialogs/startup/startup-smart.component';

/**
 * Smart component for the Session Timed Out dialog.
 * Orchestrates reconnection attempts, progress indicator display, and navigation to startup.
 */
@Component({
  selector: 'khi-session-timed-out-smart',
  imports: [SessionTimedOutLayoutComponent],
  templateUrl: './session-timed-out-smart.component.html',
  styleUrls: ['./session-timed-out-smart.component.scss'],
})
export class SessionTimedOutSmartComponent {
  private readonly dialogRef =
    inject<MatDialogRef<SessionTimedOutSmartComponent, void>>(MatDialogRef);
  private readonly dialog = inject(MatDialog);
  private readonly workbenchClient = inject(WorkbenchClientService);
  private readonly progressDialog = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );

  /**
   * Holds the error message if the previous reconnection attempt failed.
   */
  protected readonly errorMessage = signal<string | null>(null);

  /**
   * Tracks whether a reconnection operation is currently running.
   */
  protected readonly isReconnecting = this.workbenchClient.isReopening;

  /**
   * Handles user confirmation to reconnect the session on the backend.
   */
  protected async onReconnect(): Promise<void> {
    this.progressDialog.show();
    this.progressDialog.updateProgress({
      message: 'Reconnecting to server...',
      percent: 0,
      mode: 'indeterminate',
    });

    try {
      await this.workbenchClient.reopenWorkbench((msg, pct) => {
        this.progressDialog.updateProgress({
          message: msg,
          percent: pct,
          mode: 'determinate',
        });
      });
      this.dialogRef.close();
    } catch (err) {
      console.error('[SessionTimedOut] Failed to reconnect:', err);
      const message = err instanceof Error ? err.message : String(err);
      this.errorMessage.set(message);
    } finally {
      this.progressDialog.dismiss();
    }
  }

  /**
   * Handles user action to return to the startup screen and resets the current session.
   */
  protected onReturnToStartup(): void {
    this.workbenchClient.closeWorkbench();
    this.dialogRef.close();
    openStartupDialog(this.dialog);
  }
}

/**
 * Opens the Session Timed Out dialog.
 *
 * @param dialog The Angular Material MatDialog service instance.
 * @param config Optional dialog configuration overrides.
 * @returns Reference to the opened dialog.
 */
export function openSessionTimedOutDialog(
  dialog: MatDialog,
  config?: MatDialogConfig,
): MatDialogRef<SessionTimedOutSmartComponent, void> {
  return dialog.open<SessionTimedOutSmartComponent, void>(
    SessionTimedOutSmartComponent,
    {
      disableClose: true,
      ...config,
    },
  );
}
