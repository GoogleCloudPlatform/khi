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

import { Injectable, OnDestroy, computed, inject, signal } from '@angular/core';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { UserIdentityService } from 'src/app/services/api/workbench/user-identity.service';
import { OpenWorkbenchResponse_Stage } from 'src/app/generated/api/v1/workbench_pb';

/**
 * Progress event callback for Workbench opening.
 */
export type WorkbenchOpenProgressCallback = (
  message: string,
  progressPercentage: number,
  stage: OpenWorkbenchResponse_Stage,
) => void;

/**
 * WorkbenchClientService manages the lifecycle and communication with the backend WorkbenchService.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkbenchClientService implements OnDestroy {
  private readonly connectClient = inject(ConnectClientService);
  private readonly userIdService = inject(UserIdentityService);

  private readonly activeWorkbenchIdSignal = signal<string | null>(null);

  /**
   * The ID of the currently active Workbench session, or null if none is open.
   */
  public readonly activeWorkbenchId = this.activeWorkbenchIdSignal.asReadonly();

  /**
   * Whether a Workbench session is currently active.
   */
  public readonly isWorkbenchActive = computed(
    () => this.activeWorkbenchIdSignal() !== null,
  );

  private heartbeatIntervalTimer: ReturnType<typeof setInterval> | null = null;
  private readonly unloadHandler = () => {
    const id = this.activeWorkbenchIdSignal();
    if (id) {
      this.closeWorkbench(id);
    }
  };

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', this.unloadHandler);
    }
  }

  /**
   * Cleans up listeners and active timers on destroy.
   */
  public ngOnDestroy(): void {
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', this.unloadHandler);
    }
    this.stopHeartbeat();
  }

  /**
   * Opens or attaches to an in-memory Workbench session on the backend, streaming progress updates.
   */
  public async openWorkbench(
    sessionId: string,
    inspectionId: string,
    onProgress?: WorkbenchOpenProgressCallback,
  ): Promise<string | undefined> {
    const userId = this.userIdService.userId;
    const responseStream = this.connectClient.workbenchClient.openWorkbench({
      userId,
      sessionId,
      inspectionId,
    });

    let workbenchId: string | undefined;

    for await (const res of responseStream) {
      if (onProgress) {
        onProgress(res.message, res.progressPercentage, res.stage);
      }
      if (res.stage === OpenWorkbenchResponse_Stage.READY && res.workbenchId) {
        workbenchId = res.workbenchId;
      }
    }

    if (workbenchId) {
      this.activeWorkbenchIdSignal.set(workbenchId);
      this.startHeartbeat(workbenchId);
    }

    return workbenchId;
  }

  /**
   * Sends a heartbeat to refresh the TTL lease of the active Workbench.
   */
  public async heartbeat(workbenchId: string): Promise<boolean> {
    try {
      const res = await this.connectClient.workbenchClient.heartbeatWorkbench({
        workbenchId,
      });
      return res.active;
    } catch (e) {
      console.warn(`[WorkbenchClient] Heartbeat failed for ${workbenchId}:`, e);
      return false;
    }
  }

  /**
   * Closes the active Workbench session on the backend immediately.
   */
  public async closeWorkbench(workbenchId?: string): Promise<void> {
    const id = workbenchId ?? this.activeWorkbenchIdSignal();
    this.stopHeartbeat();
    this.activeWorkbenchIdSignal.set(null);

    if (id) {
      try {
        await this.connectClient.workbenchClient.closeWorkbench({
          workbenchId: id,
        });
      } catch (e) {
        console.warn(`[WorkbenchClient] Close failed for ${id}:`, e);
      }
    }
  }

  private startHeartbeat(workbenchId: string): void {
    this.stopHeartbeat();
    // Heartbeat every 15 seconds
    this.heartbeatIntervalTimer = setInterval(() => {
      this.heartbeat(workbenchId);
    }, 15000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatIntervalTimer !== null) {
      clearInterval(this.heartbeatIntervalTimer);
      this.heartbeatIntervalTimer = null;
    }
  }
}
