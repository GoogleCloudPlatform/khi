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

/**
 * timeline-render-types.ts
 * Defines the browser automation bridge and payload schemas exchanged between
 * the KHI backend headless renderer and the frontend timeline render page.
 */

/**
 * Request parameters for rendering a timeline view in the headless renderer page.
 */
export interface TimelineRenderParams {
  readonly inspectionId: string;
  readonly timelineQuery?: string;
  readonly timelineExclusionQuery?: string;
  readonly logQuery?: string;
  readonly excludeNoLogs?: boolean;
  readonly filterStartTimeNs?: string;
  readonly filterEndTimeNs?: string;
  readonly viewStartTimeMs?: number;
  readonly viewEndTimeMs?: number;
  readonly timelineIds?: readonly number[];
  readonly maxDepth?: number;
  readonly maxNodes?: number;
  readonly width?: number;
  readonly timezoneShiftHours?: number;
}

/**
 * Layout measurement and match count metadata returned after applying timeline render parameters.
 */
export interface TimelineRenderMeasureResult {
  readonly width: number;
  readonly height: number;
  readonly renderedCount: number;
  readonly totalMatched: number;
  readonly hasMore: boolean;
  readonly viewStartTimeMs: number;
  readonly viewEndTimeMs: number;
}

/**
 * Browser automation bridge for timeline rendering exposed on window.__khiCDPBridge.timelineRenderer.
 */
export interface KhiTimelineRendererBridge {
  /**
   * Loads inspection data if needed, applies filters, and returns the required viewport dimensions.
   */
  applyAndMeasure(
    params: TimelineRenderParams,
  ): Promise<TimelineRenderMeasureResult>;

  /**
   * Waits until WebGL and 2D canvas rendering have settled for the current viewport size.
   */
  waitForStableFrame(): Promise<void>;
}

/**
 * Container for Chrome DevTools Protocol automation bridges exposed on window.__khiCDPBridge.
 */
export interface KhiCDPBridge {
  timelineRenderer?: KhiTimelineRendererBridge;
}

declare global {
  interface Window {
    __khiCDPBridge?: KhiCDPBridge;
  }
}
