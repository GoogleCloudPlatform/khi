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
 * the KHI backend headless renderer and the frontend MCP timeline render page (/render/mcp-timeline).
 */

/**
 * Derived CSS color tokens for a named palette entry shared across timeline overlays and deck cards.
 */
export interface TimelineResolvedPaletteColors {
  readonly id: string;
  readonly main: string;
  readonly onMain: string;
  readonly soft: string;
  readonly surface: string;
}

/**
 * Visual line style for vertical timeline markers.
 */
export enum TimelineMarkerStyle {
  PIN = 'pin',
  REFERENCE = 'reference',
}

/**
 * Fill pattern for full-height or row-scoped timeline interval spans.
 */
export enum TimelineSpanPattern {
  SOLID = 'solid',
  HATCHED = 'hatched',
  OUTLINE = 'outline',
}

/**
 * Routing geometry for connectors between two timeline anchors.
 */
export enum TimelineConnectorStyle {
  CURVE = 'curve',
  STEP = 'step',
  CALIPER = 'caliper',
}

/**
 * Preferred placement of an inline callout card relative to its anchor point.
 */
export enum TimelineCalloutPlacement {
  AUTO = 'auto',
  LEFT = 'left',
  RIGHT = 'right',
  ABOVE = 'above',
  BELOW = 'below',
}

/**
 * Evidence source type rendered in the bottom timeline deck.
 */
export enum TimelineExcerptKind {
  DIFF = 'diff',
  LOG = 'log',
}

/**
 * Classification of a single rendered row inside a diff or log excerpt card.
 */
export enum TimelineExcerptLineKind {
  CONTEXT = 'context',
  ADDED = 'added',
  DELETED = 'deleted',
  LOG = 'log',
  SUMMARY = 'summary',
  OMITTED = 'omitted',
}

/**
 * Contiguous substring within an excerpt line with optional query match highlighting.
 */
export interface TimelineExcerptLinePart {
  readonly text: string;
  readonly matched?: boolean;
}

/**
 * Single rendered line or folded omission band inside an evidence deck card.
 */
export interface TimelineExcerptLine {
  readonly kind: TimelineExcerptLineKind;
  readonly oldLineNum?: number;
  readonly newLineNum?: number;
  readonly text: string;
  readonly parts?: readonly TimelineExcerptLinePart[];
  readonly omitted?: number;
}

/**
 * Resolved vertical event pin or global reference line.
 */
export interface TimelineResolvedMarker {
  readonly id: string;
  readonly badge: string;
  readonly label: string;
  readonly style: TimelineMarkerStyle;
  readonly paletteId: string;
  readonly timeMs: number;
  readonly timeLabel: string;
  readonly timelineId?: number;
  readonly logId?: number;
  readonly revisionIndex?: number;
}

/**
 * Resolved full-height or row-scoped time-range band.
 */
export interface TimelineResolvedSpan {
  readonly id: string;
  readonly label: string;
  readonly startTimeMs: number;
  readonly endTimeMs: number;
  readonly durationLabel: string;
  readonly timelineIds?: readonly number[];
  readonly pattern: TimelineSpanPattern;
  readonly paletteId: string;
}

/**
 * Resolved focal resource subtree highlight and left-gutter role badge.
 */
export interface TimelineResolvedRowHighlight {
  readonly id: string;
  readonly roleLabel?: string;
  readonly timelineIds: readonly number[];
  readonly paletteId: string;
}

/**
 * Resolved resource diff card or structured log card in the bottom evidence deck.
 */
export interface TimelineResolvedExcerptCard {
  readonly id: string;
  readonly kind: TimelineExcerptKind;
  readonly badge: string;
  readonly paletteId: string;
  readonly timeMs: number;
  readonly timeLabel: string;
  readonly timelineId: number;
  readonly resourcePath: string;
  readonly title: string;
  readonly subtitle: string;
  readonly lines: readonly TimelineExcerptLine[];
  readonly note?: string;
  readonly logTimestamp?: string;
  readonly insertId?: string;
  readonly logId?: number;
  readonly revisionIndex?: number;
}

/**
 * Resolved causal arrow or horizontal duration caliper between two timeline anchors.
 */
export interface TimelineResolvedConnector {
  readonly id: string;
  readonly badge?: string;
  readonly label?: string;
  readonly style: TimelineConnectorStyle;
  readonly paletteId: string;
  readonly fromTimeMs: number;
  readonly fromTimelineId?: number;
  readonly toTimeMs: number;
  readonly toTimelineId?: number;
}

/**
 * Resolved compact single-line inline callout card on the timeline chart.
 */
export interface TimelineResolvedCallout {
  readonly id: string;
  readonly label: string;
  readonly placement: TimelineCalloutPlacement;
  readonly dashed?: boolean;
  readonly paletteId: string;
  readonly timeMs: number;
  readonly timelineId?: number;
  readonly logId?: number;
  readonly revisionIndex?: number;
}

/**
 * Resolved visual overlay payload rendered over the timeline stage and bottom evidence deck.
 */
export interface TimelineSceneOverlay {
  readonly autoIncludeTimelineIds?: readonly number[];
  readonly palette?: readonly TimelineResolvedPaletteColors[];
  readonly markers?: readonly TimelineResolvedMarker[];
  readonly spans?: readonly TimelineResolvedSpan[];
  readonly rowHighlights?: readonly TimelineResolvedRowHighlight[];
  readonly dimUnlistedRows?: boolean;
  readonly excerpts?: readonly TimelineResolvedExcerptCard[];
  readonly connectors?: readonly TimelineResolvedConnector[];
  readonly callouts?: readonly TimelineResolvedCallout[];
}

/**
 * Request parameters for rendering a timeline view in the headless or interactive renderer page.
 */
export interface TimelineRenderParams {
  readonly sceneId?: string;
  readonly parentSceneId?: string;
  readonly expiresAt?: string;
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
  readonly timezoneShiftHours?: number;
  readonly overlay?: TimelineSceneOverlay;
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
 * Browser automation bridge for MCP timeline rendering exposed on window.__khiCDPBridge.timelineRenderer.
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
