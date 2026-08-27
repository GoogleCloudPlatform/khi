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
import { ConnectError, Code } from '@connectrpc/connect';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { UserIdentityService } from 'src/app/services/api/workbench/user-identity.service';
import {
  FilterResultMode,
  OpenWorkbenchResponse_Stage,
  WatchIndexProgressResponse_IndexState,
} from 'src/app/generated/api/v1/workbench_pb';
import { GetArchitectureGraphResponse } from 'src/app/generated/api/v1/architecture_graph_pb';
import { SparseBitset } from 'src/app/generated/api/v1/sparse_bitset_pb';
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
 * Final result of the filter pipeline containing sparse bitset representations of matching items.
 */
export interface FilterTimelineResult {
  readonly timelineMode?: FilterResultMode;
  readonly timelineBitset?: SparseBitset;
  readonly logMode?: FilterResultMode;
  readonly logBitset?: SparseBitset;
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

  private currentSessionId: string | null = null;
  private currentInspectionId: string | null = null;
  private inFlightReopenPromise: Promise<string | undefined> | null = null;

  private readonly isWorkbenchExpiredSignal = signal<boolean>(false);
  private readonly isReopeningSignal = signal<boolean>(false);

  private readonly indexStateSignal =
    signal<WatchIndexProgressResponse_IndexState>(
      WatchIndexProgressResponse_IndexState.UNSPECIFIED,
    );
  private readonly indexProgressPercentageSignal = signal<number>(0);
  private readonly indexMessageSignal = signal<string>('');
  private indexProgressAbortController: AbortController | null = null;

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

  /**
   * Whether the active Workbench session has expired on the backend.
   */
  public readonly isWorkbenchExpired =
    this.isWorkbenchExpiredSignal.asReadonly();

  /**
   * Whether a Workbench reopen operation is currently in progress.
   */
  public readonly isReopening = this.isReopeningSignal.asReadonly();

  /**
   * Current index construction state.
   */
  public readonly indexState = this.indexStateSignal.asReadonly();

  /**
   * Current index construction progress percentage (0 - 100).
   */
  public readonly indexProgressPercentage =
    this.indexProgressPercentageSignal.asReadonly();

  /**
   * Current index construction status message.
   */
  public readonly indexMessage = this.indexMessageSignal.asReadonly();

  /**
   * Whether the fulltext search index is currently building.
   */
  public readonly isIndexBuilding = computed(
    () =>
      this.indexStateSignal() ===
      WatchIndexProgressResponse_IndexState.BUILDING,
  );

  /**
   * Whether the fulltext search index is ready.
   */
  public readonly isIndexReady = computed(
    () =>
      this.indexStateSignal() === WatchIndexProgressResponse_IndexState.READY,
  );

  private heartbeatIntervalTimer: ReturnType<typeof setInterval> | null = null;
  private readonly unloadHandler = () => {
    const id = this.activeWorkbenchIdSignal();
    if (id) {
      this.closeWorkbench(id);
    }
  };

  private readonly visibilityChangeHandler = () => {
    if (typeof document === 'undefined') {
      return;
    }
    const id = this.activeWorkbenchIdSignal();
    if (document.visibilityState === 'hidden') {
      this.stopHeartbeat();
    } else if (
      document.visibilityState === 'visible' &&
      id &&
      !this.isWorkbenchExpiredSignal()
    ) {
      this.startHeartbeat(id);
      void this.heartbeat(id);
    }
  };

  private readonly focusHandler = () => {
    const id = this.activeWorkbenchIdSignal();
    if (id && !this.isWorkbenchExpiredSignal()) {
      void this.heartbeat(id);
    }
  };

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', this.unloadHandler);
      window.addEventListener('focus', this.focusHandler);
    }
    if (typeof document !== 'undefined') {
      document.addEventListener(
        'visibilitychange',
        this.visibilityChangeHandler,
      );
    }
  }

  /**
   * Cleans up listeners and active timers on destroy.
   */
  public ngOnDestroy(): void {
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', this.unloadHandler);
      window.removeEventListener('focus', this.focusHandler);
    }
    if (typeof document !== 'undefined') {
      document.removeEventListener(
        'visibilitychange',
        this.visibilityChangeHandler,
      );
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
    this.currentSessionId = sessionId;
    this.currentInspectionId = inspectionId;

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
      if (
        this.currentSessionId !== sessionId ||
        this.currentInspectionId !== inspectionId
      ) {
        void this.closeWorkbench(workbenchId);
        return undefined;
      }
      this.structYamlCache.clear();
      this.isWorkbenchExpiredSignal.set(false);
      this.activeWorkbenchIdSignal.set(workbenchId);
      this.startHeartbeat(workbenchId);
      if (this.indexProgressAbortController) {
        this.indexProgressAbortController.abort();
      }
      this.indexProgressAbortController = new AbortController();
      void this.watchIndexProgress(
        workbenchId,
        this.indexProgressAbortController.signal,
      );
    }

    return workbenchId;
  }

  /**
   * Watches the index progress stream for the given workbenchId, reconnecting upon 30s cycle
   * termination until the index becomes READY or FAILED, or until aborted.
   */
  public async watchIndexProgress(
    workbenchId: string,
    abortSignal?: AbortSignal,
  ): Promise<void> {
    while (!abortSignal?.aborted) {
      try {
        const responseStream =
          this.connectClient.workbenchClient.watchIndexProgress(
            { workbenchId },
            { signal: abortSignal },
          );

        if (!responseStream) {
          return;
        }

        for await (const res of responseStream) {
          if (abortSignal?.aborted) {
            return;
          }
          this.indexStateSignal.set(res.state);
          this.indexProgressPercentageSignal.set(res.progressPercentage);
          this.indexMessageSignal.set(res.message);

          if (
            res.state === WatchIndexProgressResponse_IndexState.READY ||
            res.state === WatchIndexProgressResponse_IndexState.FAILED
          ) {
            return;
          }
        }
      } catch (e) {
        if (abortSignal?.aborted) {
          return;
        }
        this.handleSessionError(e);
        if (this.isWorkbenchExpiredSignal()) {
          return;
        }
        console.warn(
          `[WorkbenchClient] WatchIndexProgress failed for ${workbenchId}:`,
          e,
        );
        // Wait 1 second before reconnecting after an unexpected network or stream failure
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    }
  }

  /**
   * Marks the workbench session as expired and stops heartbeat checks.
   */
  private markSessionExpired(): void {
    if (!this.isWorkbenchExpiredSignal()) {
      this.isWorkbenchExpiredSignal.set(true);
      this.stopHeartbeat();
    }
  }

  /**
   * Checks if an error indicates that the backend session has expired (Code.NotFound),
   * and marks the session as expired if so.
   */
  private handleSessionError(e: unknown): void {
    if (e instanceof ConnectError && e.code === Code.NotFound) {
      this.markSessionExpired();
    }
  }

  /**
   * Sends a heartbeat to refresh the TTL lease of the active Workbench.
   */
  public async heartbeat(workbenchId: string): Promise<boolean> {
    try {
      const res = await this.connectClient.workbenchClient.heartbeatWorkbench({
        workbenchId,
      });
      if (!res.active) {
        this.markSessionExpired();
        return false;
      }
      return true;
    } catch (e) {
      console.warn(`[WorkbenchClient] Heartbeat failed for ${workbenchId}:`, e);
      this.handleSessionError(e);
      return false;
    }
  }

  /**
   * Reopens an expired Workbench session using the stored session and inspection identifiers.
   * Concurrent invocations are deduplicated to share a single in-flight reopening promise.
   *
   * @param onProgress Optional callback for reporting reopening progress stages.
   * @returns The restored Workbench ID.
   */
  public async reopenWorkbench(
    onProgress?: WorkbenchOpenProgressCallback,
  ): Promise<string | undefined> {
    if (!this.currentSessionId || !this.currentInspectionId) {
      throw new Error(
        'No active or previous inspection session available to reopen.',
      );
    }

    if (this.inFlightReopenPromise) {
      return await this.inFlightReopenPromise;
    }

    this.isReopeningSignal.set(true);
    this.inFlightReopenPromise = (async () => {
      try {
        const workbenchId = await this.openWorkbench(
          this.currentSessionId!,
          this.currentInspectionId!,
          onProgress,
        );
        return workbenchId;
      } finally {
        this.isReopeningSignal.set(false);
        this.inFlightReopenPromise = null;
      }
    })();

    return await this.inFlightReopenPromise;
  }

  /**
   * Closes the active Workbench session on the backend immediately.
   */
  public async closeWorkbench(workbenchId?: string): Promise<void> {
    const id = workbenchId ?? this.activeWorkbenchIdSignal();
    if (this.indexProgressAbortController) {
      this.indexProgressAbortController.abort();
      this.indexProgressAbortController = null;
    }
    this.stopHeartbeat();
    this.structYamlCache.clear();
    this.currentSessionId = null;
    this.currentInspectionId = null;
    this.isWorkbenchExpiredSignal.set(false);
    this.isReopeningSignal.set(false);
    this.activeWorkbenchIdSignal.set(null);
    this.indexStateSignal.set(
      WatchIndexProgressResponse_IndexState.UNSPECIFIED,
    );
    this.indexProgressPercentageSignal.set(0);
    this.indexMessageSignal.set('');

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

    try {
      const res = await this.connectClient.workbenchClient.readStructYAML({
        workbenchId,
        structId,
      });
      const yaml = res.yaml ?? '';
      this.structYamlCache.put(structId, yaml);
      return yaml;
    } catch (e) {
      this.handleSessionError(e);
      throw e;
    }
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

    try {
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

      let result: FilterTimelineResult = {};

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
            timelineMode: res.payload.value.timelineMode,
            timelineBitset: res.payload.value.timelineBitset,
            logMode: res.payload.value.logMode,
            logBitset: res.payload.value.logBitset,
          };
        }
      }

      return result;
    } catch (e) {
      this.handleSessionError(e);
      throw e;
    }
  }

  /**
   * Fetches the architecture graph at the specified timestamp from the active Workbench session.
   *
   * @param timestampNs The timestamp in nanoseconds.
   * @param timelineBitset Optional sparse bitset of allowed timeline IDs.
   * @param deletionThresholdSeconds Optional deletion threshold in seconds.
   * @param signal Optional AbortSignal to cancel the RPC.
   * @returns The architecture graph response containing nodes, pods, services, owners, and edges.
   */
  public async getArchitectureGraph(
    timestampNs: bigint,
    timelineBitset?: SparseBitset,
    deletionThresholdSeconds?: number,
    signal?: AbortSignal,
  ): Promise<GetArchitectureGraphResponse> {
    const workbenchId = this.activeWorkbenchIdSignal();
    if (!workbenchId) {
      throw new Error('No active Workbench session found.');
    }

    return await this.connectClient.workbenchClient.getArchitectureGraph(
      {
        workbenchId,
        timestampNs,
        timelineBitset,
        deletionThresholdSeconds,
      },
      { signal },
    );
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
