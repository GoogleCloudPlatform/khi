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
import { LRUCache } from 'src/app/common/lru-cache';

/**
 * Progress event callback for Workbench opening.
 */
export type WorkbenchOpenProgressCallback = (
  message: string,
  progressPercentage: number,
  stage: OpenWorkbenchResponse_Stage,
) => void;

/**
 * Parameters for the timeline and log filter pipeline.
 */
export interface FilterTimelineParams {
  readonly timelineQuery?: string;
  readonly timelineExclusionQuery?: string;
  readonly logQuery?: string;
  readonly excludeNoLogs?: boolean;
}

/**
 * Filter progress callback for streaming updates.
 */
export type FilterProgressCallback = (
  stageName: string,
  current: number,
  total: number,
) => void;

/**
 * Final result of the filter pipeline.
 */
export interface FilterTimelineResult {
  readonly timelineIds: readonly number[];
  readonly logIds: readonly number[];
}

/**
 * WorkbenchClientService manages the lifecycle and communication with the backend WorkbenchService.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkbenchClientService implements OnDestroy {
  /**
   * Maximum number of interned struct YAML strings to cache in memory.
   */
  private static readonly STRUCT_YAML_CACHE_CAPACITY = 2000;

  private readonly connectClient = inject(ConnectClientService);
  private readonly userIdService = inject(UserIdentityService);

  private readonly activeWorkbenchIdSignal = signal<string | null>(null);
  private readonly structYamlCache = new LRUCache<number, string>(
    WorkbenchClientService.STRUCT_YAML_CACHE_CAPACITY,
  );

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
      this.structYamlCache.clear();
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
    this.structYamlCache.clear();
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

  /**
   * Fetches the decoded YAML representation of an interned struct by ID from the active Workbench session.
   *
   * @param structId The interned struct ID to decode.
   * @returns The YAML string representation.
   */
  public async readStructYAML(structId: number): Promise<string> {
    if (!structId || structId <= 0) {
      return '';
    }

    const cached = this.structYamlCache.get(structId);
    if (cached !== undefined) {
      return cached;
    }

    const workbenchId = this.activeWorkbenchIdSignal();
    if (!workbenchId) {
      throw new Error('No active Workbench session found.');
    }
    const res = await this.connectClient.workbenchClient.readStructYAML({
      workbenchId,
      structId,
    });
    const yaml = res.yaml ?? '';
    this.structYamlCache.put(structId, yaml);
    return yaml;
  }

  /**
   * Evaluates the timeline and log filter pipeline on the backend Workbench session and returns the final filtered IDs.
   *
   * @param params The search queries and options for the pipeline.
   * @param onProgress Optional callback invoked when progress updates are streamed from the server.
   * @param signal Optional AbortSignal to cancel the streaming RPC.
   * @returns The filtered timeline IDs and log IDs.
   */
  public async filterTimeline(
    params: FilterTimelineParams,
    onProgress?: FilterProgressCallback,
    signal?: AbortSignal,
  ): Promise<FilterTimelineResult> {
    const workbenchId = this.activeWorkbenchIdSignal();
    if (!workbenchId) {
      throw new Error('No active Workbench session found.');
    }

    const responseStream = this.connectClient.workbenchClient.filterTimeline(
      {
        workbenchId,
        timelineQuery: params.timelineQuery ?? '',
        timelineExclusionQuery: params.timelineExclusionQuery ?? '',
        logQuery: params.logQuery ?? '',
        excludeNoLogs: params.excludeNoLogs ?? false,
      },
      { signal },
    );

    let result: FilterTimelineResult = {
      timelineIds: [],
      logIds: [],
    };

    for await (const res of responseStream) {
      if (res.payload.case === 'progress' && res.payload.value) {
        if (onProgress) {
          onProgress(
            res.payload.value.stageName ?? '',
            res.payload.value.current ?? 0,
            res.payload.value.total ?? 0,
          );
        }
      } else if (res.payload.case === 'result' && res.payload.value) {
        result = {
          timelineIds: res.payload.value.timelineIds ?? [],
          logIds: res.payload.value.logIds ?? [],
        };
      }
    }

    return result;
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
