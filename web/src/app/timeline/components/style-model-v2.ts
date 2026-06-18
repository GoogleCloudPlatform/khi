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

import {
  HDRColor4,
  RendererConvertUtil,
} from 'src/app/timeline/components/canvas/convertutil';
import { TickImportance } from 'src/app/timeline/components/timeline-ruler.viewmodel';
import {
  TimelineChartItemHighlightType,
  TimelineHighlightType,
} from 'src/app/timeline/components/interaction-model';
import { RevisionStateStyle, Severity } from 'src/app/store/domain/style';
import { StyleStoreLike } from 'src/app/store/domain/style-store';

/**
 * Baseline height of a timeline row in pixels.
 */
export const BASE_ROW_HEIGHT = 25.0;

/**
 * Baseline thickness of a horizontal border line in pixels.
 */
export const BASE_HORIZONTAL_BORDER_THICKNESS = 1.0;

/**
 * Minimum and maximum thickness of a horizontal border line in pixels.
 */
export const HORIZONTAL_BORDER_THICKNESS_RANGE = [0.2, 2.0];

/**
 * Style configuration for the timeline ruler.
 */
export interface TimelineRulerStyle {
  /**
   * Order of severities to be drawn in the histogram.
   * Severities appearing later in the array are drawn on top of earlier ones.
   */
  readonly severitiesInDrawOrder: readonly Severity[];
  /**
   * Colors for each severity level in the histogram.
   */
  readonly severityColors: { [severityId: number]: HDRColor4 };
  /**
   * Stroke colors for each severity level (currently used for borders).
   */
  readonly severityStrokeColors: { [severityId: number]: HDRColor4 };
  /**
   * Alpha transparency for non-highlighted histogram bars (0-1).
   */
  readonly nonHighlightedAlpha: number;
  /**
   * Alpha transparency for highlighted histogram bars (0-1).
   */
  readonly highlightedAlpha: number;
  /**
   * Thickness of the histogram lines in pixels.
   */
  readonly histogramLineThickness: number;
  /**
   * Height of the timeline header/ruler in pixels.
   */
  readonly headerHeightInPx: number;
  /**
   * Height of the tick marks on the ruler based on their importance.
   */
  readonly heightByImportance: { readonly [key in TickImportance]: number };
  /**
   * Thickness of the tick marks on the ruler based on their importance.
   */
  readonly rulerThicknessByImportance: {
    readonly [key in TickImportance]: number;
  };
  /**
   * Extra height for ruler ticks extending into the timeline area.
   */
  readonly rulerExtraHeightByImportance: {
    readonly [key in TickImportance]: number;
  };
  /**
   * Color of the ruler lines and text.
   */
  readonly rulerColor: HDRColor4;
  /**
   * Maximum height of the histogram bars in pixels.
   */
  readonly maxHistogramHeightInPx: number;
}

/**
 * Style configuration for timeline revisions.
 */
export interface TimelineRevisionStyle {
  /**
   * Vertical padding inside the revision box in pixels.
   */
  readonly verticalPaddingInPx: number;
  /**
   * Thickness of the revision box border in pixels.
   */
  readonly borderThickness: number;
  /**
   * Padding around the text inside the revision box [x, y].
   */
  readonly fontPaddingInPx: readonly [number, number];
  /**
   * Font size in pixels.
   */
  readonly fontSizeInPx: number;
  /**
   * SDF font thickness/weight adjustment based on selection state.
   */
  readonly fontThicknessBySelectionType: {
    readonly [key in TimelineChartItemHighlightType]: number;
  };
  /**
   * Padding around the icon [x, y].
   */
  readonly iconPaddingInPx: readonly [number, number];
  /**
   * Size of the icon in pixels.
   */
  readonly iconSizeInPx: number;
  /**
   * SDF icon thickness/weight adjustment based on selection state.
   */
  readonly iconThicknessBySelectionType: {
    readonly [key in TimelineChartItemHighlightType]: number;
  };
  /**
   * Font antialiasing factor (smoothing).
   */
  readonly fontAntialias: number;
  /**
   * Icon antialiasing factor (smoothing).
   */
  readonly iconAntialias: number;
  /**
   * Minimum step (width) required to render text in pixels.
   */
  readonly fontStepInPx: number;
  /**
   * Thickness of the selection border highlight.
   */
  readonly selectionBorderThickness: number;
  /**
   * Thickness of the hover border highlight.
   */
  readonly hoverBorderThickness: number;
}

/**
 * Style configuration for distinct revision states (Normal, Deleted, PartialInfo).
 */
export interface TimelineRevisionStateStyle {
  /**
   * Alpha transparency of the revision state pattern (0-1).
   */
  readonly alphaTransparency: number;
  /**
   * Pattern coefficient for the border stripe.
   */
  readonly borderStripePattern: number;
  /**
   * Pattern coefficient for the body stripe.
   */
  readonly bodyStripePattern: number;
}

/**
 * Style configuration for timeline events.
 */
export interface TimelineEventStyle {
  /**
   * Vertical padding for the event indicator in pixels.
   */
  readonly verticalPaddingInPx: number;
  /**
   * Ratio of the event height that is colored by severity.
   */
  readonly severityColorRatio: number;
  /**
   * Thickness of the event border.
   */
  readonly borderThickness: number;
  /**
   * Antialiasing factor for the border.
   */
  readonly borderAntialias: number;
  /**
   * Thickness of the selection border.
   */
  readonly selectionBorderThickness: number;
  /**
   * Thickness of the hover border.
   */
  readonly hoverBorderThickness: number;
}

/**
 * Style configuration for the entire timeline chart.
 */
export interface TimelineChartStyle {
  /**
   * Tint color applied to timeline items based on their highlight state.
   */
  readonly timelineTintColorByHighlightType: {
    readonly [key in TimelineHighlightType]: HDRColor4;
  };
  /**
   * Style configuration for different revision states.
   */
  readonly revisionStateStyle: {
    readonly [key in RevisionStateStyle]: TimelineRevisionStateStyle;
  };
  /**
   * Color of the horizontal separator lines.
   */
  readonly horizontalLineColor: HDRColor4;
  /**
   * Color overlay for areas outside the log period.
   */
  readonly outsideOfLogPeriodColor: HDRColor4;
  /**
   * Color of the selection border.
   */
  readonly selectionBorderColor: HDRColor4;
  /**
   * Color of the hover border.
   */
  readonly highlightBorderColor: HDRColor4;
  /**
   * Pitch (spacing) of the border stripe pattern.
   */
  readonly borderStripePitch: number;
  /**
   * Pitch (spacing) of the body stripe pattern.
   */
  readonly bodyStripePitch: number;
}

/**
 * Helper to dynamically derive a TimelineRevisionStyle configuration from row height.
 */
export function getRevisionStyleForHeight(
  height: number,
): TimelineRevisionStyle {
  const verticalPaddingInPx = Math.max(1.0, Math.round(height * 0.05));
  const borderThickness = Math.max(1.0, Math.round(height * 0.09));
  const fontPaddingY = Math.max(2.0, Math.round(height * 0.18));
  const fontPaddingInPx: [number, number] = [8, fontPaddingY];
  const fontSizeInPx = Math.max(8.0, Math.round(height * 0.4));
  const iconSizeInPx = Math.max(8.0, Math.round(height * 0.47));
  const iconPaddingY = Math.max(2.0, Math.round(height * 0.18));
  const iconPaddingInPx: [number, number] = [6, iconPaddingY];
  const fontStepInPx = Math.max(4.0, Math.round(height * 0.27));
  const selectionBorderThickness = Math.max(3.0, Math.round(height * 0.2));
  const hoverBorderThickness = selectionBorderThickness * 0.8;

  return {
    verticalPaddingInPx,
    borderThickness,
    fontPaddingInPx,
    fontSizeInPx,
    fontThicknessBySelectionType: {
      [TimelineChartItemHighlightType.None]: 0.7,
      [TimelineChartItemHighlightType.Selected]: 0.4,
      [TimelineChartItemHighlightType.Hovered]: 0.2,
    },
    iconSizeInPx,
    iconPaddingInPx,
    iconThicknessBySelectionType: {
      [TimelineChartItemHighlightType.None]: 0.5,
      [TimelineChartItemHighlightType.Selected]: 0.4,
      [TimelineChartItemHighlightType.Hovered]: 0.4,
    },
    fontAntialias: 0.2,
    iconAntialias: 0.2,
    fontStepInPx,
    selectionBorderThickness,
    hoverBorderThickness,
  };
}

/**
 * Helper to dynamically derive a TimelineEventStyle configuration from row height.
 */
export function getEventStyleForHeight(height: number): TimelineEventStyle {
  const verticalPaddingInPx = Math.max(
    2.0,
    height * 0.2 + (height === 30 ? 0.5 : 0.0),
  );
  const selectionBorderThickness = Math.max(2.0, Math.round(height * 0.13));

  return {
    verticalPaddingInPx,
    severityColorRatio: 0.55,
    borderThickness: 4,
    borderAntialias: 0.03,
    selectionBorderThickness,
    hoverBorderThickness: 2,
  };
}

/**
 * Generates the default style configuration for the timeline chart.
 */
export function generateDefaultChartStyle(): TimelineChartStyle {
  return {
    timelineTintColorByHighlightType: {
      [TimelineHighlightType.None]:
        RendererConvertUtil.hexSRGBToHDRColor('#00000000'),
      [TimelineHighlightType.Selected]: [0.8, 0.91, 0.81, 0.96],
      [TimelineHighlightType.Hovered]: [0.8, 0.91, 0.81, 0.6],
      [TimelineHighlightType.ChildrenOfSelected]: [0.95, 1, 0.95, 0.8],
    },
    horizontalLineColor: RendererConvertUtil.hexSRGBToHDRColor('#333333FF'),
    outsideOfLogPeriodColor: RendererConvertUtil.hexSRGBToHDRColor('#00000055'),
    selectionBorderColor: RendererConvertUtil.hexSRGBToHDRColor('#FFFF22FF'),
    highlightBorderColor: RendererConvertUtil.hexSRGBToHDRColor('#FFFF22FF'),
    revisionStateStyle: {
      [RevisionStateStyle.NORMAL]: {
        alphaTransparency: 0.4,
        borderStripePattern: 0,
        bodyStripePattern: 0,
      },
      [RevisionStateStyle.DELETED]: {
        alphaTransparency: 0.4,
        borderStripePattern: 1,
        bodyStripePattern: 0,
      },
      [RevisionStateStyle.PARTIAL_INFO]: {
        alphaTransparency: 0.4,
        borderStripePattern: 0,
        bodyStripePattern: 1,
      },
    },
    borderStripePitch: 5,
    bodyStripePitch: 20,
  };
}

/**
 * Generates the default style configuration for the timeline ruler.
 */
export function generateDefaultRulerStyle(
  styleStore?: StyleStoreLike,
): TimelineRulerStyle {
  const severities = styleStore?.severities ?? [];
  const severityColors: { [severityId: number]: HDRColor4 } = {};
  const severityStrokeColors: { [severityId: number]: HDRColor4 } = {};

  for (const s of severities) {
    severityColors[s.id] = [
      s.backgroundColor.r,
      s.backgroundColor.g,
      s.backgroundColor.b,
      s.backgroundColor.a,
    ];
    severityStrokeColors[s.id] = [
      s.foregroundColor.r,
      s.foregroundColor.g,
      s.foregroundColor.b,
      s.foregroundColor.a,
    ];
  }

  return {
    severitiesInDrawOrder: [...severities].sort((a, b) => a.order - b.order),
    severityColors,
    severityStrokeColors,
    nonHighlightedAlpha: 0.2,
    highlightedAlpha: 1,
    histogramLineThickness: 0.5,
    headerHeightInPx: 60,
    heightByImportance: {
      [TickImportance.Low]: 10,
      [TickImportance.Middle]: 20,
      [TickImportance.High]: 30,
    },
    rulerThicknessByImportance: {
      [TickImportance.Low]: 0.25,
      [TickImportance.Middle]: 0.5,
      [TickImportance.High]: 1,
    },
    rulerExtraHeightByImportance: {
      [TickImportance.Low]: 10,
      [TickImportance.Middle]: 20,
      [TickImportance.High]: 30,
    },
    rulerColor: RendererConvertUtil.hexSRGBToHDRColor('#888888'),
    maxHistogramHeightInPx: 30,
  };
}
