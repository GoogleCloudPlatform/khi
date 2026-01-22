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

import * as generated from 'src/app/generated';
import { TimelineLayer } from 'src/app/store/timeline';
import { TickImportance } from './timeline-ruler.viewmodel';
import {
  TimelineChartItemHighlightType,
  TimelineHighlightType,
} from './interaction-model';
import { HDRColor4, RendererConvertUtil } from './canvas/convertutil';

/**
 * Style configuration for the timeline ruler.
 */
export interface TimelineRulerStyle {
  /**
   * Order of severities to be drawn in the histogram.
   * Severities appearing later in the array are drawn on top of earlier ones.
   */
  severitiesInDrawOrder: generated.Severity[];
  /**
   * Colors for each severity level in the histogram.
   */
  severityColors: { [key in generated.Severity]: HDRColor4 };
  /**
   * Stroke colors for each severity level (currently used for borders).
   */
  severityStrokeColors: { [key in generated.Severity]: HDRColor4 };
  /**
   * Alpha transparency for non-highlighted histogram bars (0-1).
   */
  nonHighlightedAlpha: number;
  /**
   * Alpha transparency for highlighted histogram bars (0-1).
   */
  highlightedAlpha: number;
  /**
   * Thickness of the histogram lines in pixels.
   */
  histogramLineThickness: number;
  /**
   * Height of the timeline header/ruler in pixels.
   */
  headerHeightInPx: number;
  /**
   * Height of the tick marks on the ruler based on their importance.
   */
  heightByImportance: { [key in TickImportance]: number };
  /**
   * Thickness of the tick marks on the ruler based on their importance.
   */
  rulerThicknessByImportance: { [key in TickImportance]: number };
  /**
   * Extra height for ruler ticks extending into the timeline area.
   */
  rulerExtraHeightByImportance: { [key in TickImportance]: number };
  /**
   * Color of the ruler lines and text.
   */
  rulerColor: HDRColor4;
  /**
   * Maximum height of the histogram bars in pixels.
   */
  maxHistogramHeightInPx: number;
}

/**
 * Style configuration for timeline revisions.
 */
export interface TimelineRevisionStyle {
  /**
   * Vertical padding inside the revision box in pixels.
   */
  verticalPaddingInPx: number;
  /**
   * Thickness of the revision box border in pixels.
   */
  borderThickness: number;
  /**
   * Padding around the text inside the revision box [x, y].
   */
  fontPaddingInPx: [number, number];
  /**
   * Font size in pixels.
   */
  fontSizeInPx: number;
  /**
   * SDF font thickness/weight adjustment based on selection state.
   */
  fontThicknessBySelectionType: {
    [key in TimelineChartItemHighlightType]: number;
  };
  /**
   * Padding around the icon [x, y].
   */
  iconPaddingInPx: [number, number];
  /**
   * Size of the icon in pixels.
   */
  iconSizeInPx: number;
  /**
   * SDF icon thickness/weight adjustment based on selection state.
   */
  iconThicknessBySelectionType: {
    [key in TimelineChartItemHighlightType]: number;
  };
  /**
   * Font antialiasing factor (smoothing).
   */
  fontAntialias: number;
  /**
   * Icon antialiasing factor (smoothing).
   */
  iconAntialias: number;
  /**
   * Minimum step (width) required to render text in pixels.
   */
  fontStepInPx: number;
  /**
   * Thickness of the selection border highlight.
   */
  selectionBorderThickness: number;
  /**
   * Thickness of the hover border highlight.
   */
  hoverBorderThickness: number;
}

/**
 * Style configuration for distinct revision states (Normal, Deleted, PartialInfo).
 */
export interface TimelineRevisionStateStyle {
  /**
   * Alpha transparency of the revision state pattern (0-1).
   */
  alphaTransparency: number;
  /**
   * Pattern coefficient for the border stripe.
   */
  borderStripePatten: number;
  /**
   * Pattern coefficient for the body stripe.
   */
  bodyStripePattern: number;
}

/**
 * Style configuration for timeline events.
 */
export interface TimelineEventStyle {
  /**
   * Vertical padding for the event indicator in pixels.
   */
  verticalPaddingInPx: number;
  /**
   * Ratio of the event height that is colored by severity.
   */
  severityColorRatio: number;
  /**
   * Thickness of the event border.
   */
  borderThickness: number;
  /**
   * Antialiasing factor for the border.
   */
  borderAntialias: number;
  /**
   * Thickness of the selection border.
   */
  selectionBorderThickness: number;
  /**
   * Thickness of the hover border.
   */
  hoverBorderThickness: number;
}

/**
 * Style configuration for the entire timeline chart, including layers, lines, and item styles.
 */
export interface TimelineChartStyle {
  /**
   * Height of each timeline layer row in pixels.
   */
  heightsByLayer: { [key in TimelineLayer]: number };
  /**
   * Thickness of the horizontal separator lines between layers.
   */
  horizontalLineThicknessByLayer: { [key in TimelineLayer]: number };
  /**
   * Background color for each timeline layer.
   */
  timelineBackgroundColorByLayer: { [key in TimelineLayer]: HDRColor4 };
  /**
   * Tint color applied to timeline items based on their highlight state (hover, selected, etc.).
   */
  timelineTintColorByHighlightType: {
    [key in TimelineHighlightType]: HDRColor4;
  };
  /**
   * Style configuration for revisions within each layer.
   */
  revisionStylesByLayer: { [key in TimelineLayer]: TimelineRevisionStyle };
  /**
   * Style configuration for events within each layer.
   */
  eventStylesByLayer: { [key in TimelineLayer]: TimelineEventStyle };
  /**
   * Style configuration for different revision states (e.g., specific stripe patterns).
   */
  revisionStateStyle: {
    [key in generated.RevisionStateStyle]: TimelineRevisionStateStyle;
  };
  /**
   * Color of the horizontal separator lines.
   */
  horizontalLineColor: HDRColor4;
  /**
   * Color overlay for areas outside the log period.
   */
  outsideOfLogPeriodColor: HDRColor4;
  /**
   * Color of the selection border.
   */
  selectionBorderColor: HDRColor4;
  /**
   * Color of the hover border.
   */
  highlightBorderColor: HDRColor4;
  /**
   * Pitch (spacing) of the border stripe pattern.
   */
  borderStripePitch: number;
  /**
   * Pitch (spacing) of the body stripe pattern.
   */
  bodyStripePitch: number;
}

const dummyTimelineRevisionStyle: TimelineRevisionStyle = {
  verticalPaddingInPx: 0,
  borderThickness: 0,
  fontPaddingInPx: [0, 0],
  fontSizeInPx: 0,
  fontThicknessBySelectionType: {
    [TimelineChartItemHighlightType.None]: 0,
    [TimelineChartItemHighlightType.Selected]: 0,
    [TimelineChartItemHighlightType.Hovered]: 0,
  },
  fontAntialias: 0,
  iconAntialias: 0,
  iconPaddingInPx: [0, 0],
  iconSizeInPx: 0,
  iconThicknessBySelectionType: {
    [TimelineChartItemHighlightType.None]: 0,
    [TimelineChartItemHighlightType.Selected]: 0,
    [TimelineChartItemHighlightType.Hovered]: 0,
  },
  fontStepInPx: 0,
  selectionBorderThickness: 0,
  hoverBorderThickness: 0,
};

const dummyTimelineEventStyle: TimelineEventStyle = {
  verticalPaddingInPx: 0,
  severityColorRatio: 0,
  borderThickness: 0,
  borderAntialias: 0,
  selectionBorderThickness: 0,
  hoverBorderThickness: 0,
};

/**
 * Generates the default style configuration for the timeline chart.
 */
export function generateDefaultChartStyle(): TimelineChartStyle {
  return {
    heightsByLayer: {
      [TimelineLayer.APIVersion]: 0, // No timeline only for API version
      [TimelineLayer.Kind]: 25,
      [TimelineLayer.Namespace]: 25,
      [TimelineLayer.Name]: 30,
      [TimelineLayer.Subresource]: 22,
    },
    horizontalLineThicknessByLayer: {
      [TimelineLayer.APIVersion]: 0,
      [TimelineLayer.Kind]: 0.5,
      [TimelineLayer.Namespace]: 0.5,
      [TimelineLayer.Name]: 0.5,
      [TimelineLayer.Subresource]: 0.25,
    },
    timelineBackgroundColorByLayer: {
      [TimelineLayer.APIVersion]:
        RendererConvertUtil.hexSRGBToHDRColor('#FFFFFF'),
      [TimelineLayer.Kind]: RendererConvertUtil.hexSRGBToHDRColor('#3f51b5'),
      [TimelineLayer.Namespace]:
        RendererConvertUtil.hexSRGBToHDRColor('#646464'),
      [TimelineLayer.Name]: RendererConvertUtil.hexSRGBToHDRColor('#EEEEEE'),
      [TimelineLayer.Subresource]:
        RendererConvertUtil.hexSRGBToHDRColor('#FFFFFF'),
    },
    timelineTintColorByHighlightType: {
      [TimelineHighlightType.None]:
        RendererConvertUtil.hexSRGBToHDRColor('#00000000'),
      [TimelineHighlightType.Selected]: [0.8, 0.91, 0.81, 0.7],
      [TimelineHighlightType.Hovered]: [0.8, 0.91, 0.81, 0.4],
      [TimelineHighlightType.ChildrenOfSelected]: [0.95, 1, 0.95, 0.2],
    },
    revisionStylesByLayer: {
      [TimelineLayer.APIVersion]: dummyTimelineRevisionStyle,
      [TimelineLayer.Kind]: dummyTimelineRevisionStyle,
      [TimelineLayer.Namespace]: dummyTimelineRevisionStyle,
      [TimelineLayer.Name]: {
        verticalPaddingInPx: 2,
        borderThickness: 3,
        fontPaddingInPx: [8, 6],
        fontSizeInPx: 12,
        fontThicknessBySelectionType: {
          [TimelineChartItemHighlightType.None]: 0.7,
          [TimelineChartItemHighlightType.Selected]: 0.4,
          [TimelineChartItemHighlightType.Hovered]: 0.2,
        },
        iconSizeInPx: 14,
        iconPaddingInPx: [6, 6],
        iconThicknessBySelectionType: {
          [TimelineChartItemHighlightType.None]: 0.5,
          [TimelineChartItemHighlightType.Selected]: 0.4,
          [TimelineChartItemHighlightType.Hovered]: 0.4,
        },
        fontAntialias: 0.2,
        iconAntialias: 0.2,
        fontStepInPx: 8,
        selectionBorderThickness: 6,
        hoverBorderThickness: 6 * 0.8,
      },
      [TimelineLayer.Subresource]: {
        verticalPaddingInPx: 1,
        borderThickness: 2,
        fontPaddingInPx: [8, 4],
        fontSizeInPx: 10,
        fontThicknessBySelectionType: {
          [TimelineChartItemHighlightType.None]: 0.7,
          [TimelineChartItemHighlightType.Selected]: 0.4,
          [TimelineChartItemHighlightType.Hovered]: 0.2,
        },
        iconSizeInPx: 12,
        iconPaddingInPx: [6, 4],
        iconThicknessBySelectionType: {
          [TimelineChartItemHighlightType.None]: 0.5,
          [TimelineChartItemHighlightType.Selected]: 0.4,
          [TimelineChartItemHighlightType.Hovered]: 0.4,
        },
        fontAntialias: 0.2,
        iconAntialias: 0.2,
        fontStepInPx: 6,
        selectionBorderThickness: 5,
        hoverBorderThickness: 5 * 0.8,
      },
    },
    eventStylesByLayer: {
      [TimelineLayer.APIVersion]: dummyTimelineEventStyle,
      [TimelineLayer.Kind]: dummyTimelineEventStyle,
      [TimelineLayer.Namespace]: dummyTimelineEventStyle,
      [TimelineLayer.Name]: {
        verticalPaddingInPx: 6.5,
        severityColorRatio: 0.55,
        borderThickness: 4,
        borderAntialias: 0.03,
        selectionBorderThickness: 4,
        hoverBorderThickness: 2,
      },
      [TimelineLayer.Subresource]: {
        verticalPaddingInPx: 4,
        severityColorRatio: 0.55,
        borderThickness: 4,
        borderAntialias: 0.03,
        selectionBorderThickness: 3,
        hoverBorderThickness: 2,
      },
    },
    horizontalLineColor: RendererConvertUtil.hexSRGBToHDRColor('#333333FF'),
    outsideOfLogPeriodColor: RendererConvertUtil.hexSRGBToHDRColor('#00000055'),
    selectionBorderColor: RendererConvertUtil.hexSRGBToHDRColor('#FFFF22FF'),
    highlightBorderColor: RendererConvertUtil.hexSRGBToHDRColor('#FFFF22FF'),
    revisionStateStyle: {
      [generated.RevisionStateStyle.Normal]: {
        alphaTransparency: 0.4,
        borderStripePatten: 0,
        bodyStripePattern: 0,
      },
      [generated.RevisionStateStyle.Deleted]: {
        alphaTransparency: 0.4,
        borderStripePatten: 1,
        bodyStripePattern: 0,
      },
      [generated.RevisionStateStyle.PartialInfo]: {
        alphaTransparency: 0.4,
        borderStripePatten: 0,
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
export function generateDefaultRulerStyle(): TimelineRulerStyle {
  return {
    severitiesInDrawOrder: [
      generated.Severity.SeverityUnknown,
      generated.Severity.SeverityInfo,
      generated.Severity.SeverityWarning,
      generated.Severity.SeverityError,
      generated.Severity.SeverityFatal,
    ],
    severityColors: {
      [generated.Severity.SeverityFatal]:
        generated.severityColors[
          generated.severities[generated.Severity.SeverityFatal]
        ],
      [generated.Severity.SeverityError]:
        generated.severityColors[
          generated.severities[generated.Severity.SeverityError]
        ],
      [generated.Severity.SeverityWarning]:
        generated.severityColors[
          generated.severities[generated.Severity.SeverityWarning]
        ],
      [generated.Severity.SeverityInfo]:
        generated.severityColors[
          generated.severities[generated.Severity.SeverityInfo]
        ],
      [generated.Severity.SeverityUnknown]: [0.5, 0.5, 0.5, 1], // the severity color(black) is too vivid for histogram. Use gray instead.
    },
    severityStrokeColors: {
      [generated.Severity.SeverityFatal]:
        generated.severityBorderColors[
          generated.severities[generated.Severity.SeverityFatal]
        ],
      [generated.Severity.SeverityError]:
        generated.severityBorderColors[
          generated.severities[generated.Severity.SeverityError]
        ],
      [generated.Severity.SeverityWarning]:
        generated.severityBorderColors[
          generated.severities[generated.Severity.SeverityWarning]
        ],
      [generated.Severity.SeverityInfo]:
        generated.severityBorderColors[
          generated.severities[generated.Severity.SeverityInfo]
        ],
      [generated.Severity.SeverityUnknown]: [0.8, 0.8, 0.8, 1], // the severity color(black) is too vivid for histogram. Use gray instead.
    },
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
