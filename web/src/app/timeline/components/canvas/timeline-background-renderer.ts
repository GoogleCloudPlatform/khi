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

import { TimelineRulerViewModel } from '../timeline-ruler.viewmodel';
import { TimelineChartViewModel } from '../timeline-chart.viewmodel';
import { TimelineHighlight } from '../interaction-model';
import {
  TimelineRulerStyle,
  TimelineChartStyle,
  BASE_ROW_HEIGHT,
  BASE_HORIZONTAL_BORDER_THICKNESS,
  HORIZONTAL_BORDER_THICKNESS_RANGE,
} from '../style-model-v2';
import { RendererConvertUtil } from './convertutil';

/**
 * Renderer for the timeline background.
 *
 * This class is responsible for rendering:
 * - The background color of each timeline.
 * - The grid lines (rulers) and horizontal separators.
 * - The "outside log period" dimming effect.
 * - Timeline highlights.
 */
export class TimelineBackgroundRenderer {
  private width = 0;
  private height = 0;
  private dpr = 1;

  private rulerStyle: TimelineRulerStyle | null = null;
  private chartStyle: TimelineChartStyle | null = null;

  private rulerViewModel: TimelineRulerViewModel | null = null;
  private chartViewModel: TimelineChartViewModel | null = null;

  private timelineHighlights: TimelineHighlight | null = null;

  constructor(private ctx: CanvasRenderingContext2D) {}

  /**
   * Resize the canvas and update the internal state.
   *
   * @param width - The logical width of the canvas.
   * @param height - The logical height of the canvas.
   * @param dpr - The device pixel ratio.
   */
  resize(width: number, height: number, dpr: number) {
    this.width = width;
    this.height = height;
    this.dpr = dpr;
  }

  /**
   * Render the background of the timeline.
   *
   * @param leftEdgeTime - The time at the left edge of the viewport in nanoseconds.
   * @param pixelsPerNs - The rendering scale in pixels per nanosecond.
   */
  render(leftEdgeTime: number, pixelsPerNs: number) {
    if (
      this.rulerStyle === null ||
      this.rulerViewModel === null ||
      this.chartStyle === null ||
      this.chartViewModel === null
    ) {
      return;
    }
    this.ctx.setTransform(1, 0, 0, 1, 0, 0);
    this.ctx.clearRect(0, 0, this.width * this.dpr, this.height * this.dpr);
    this.ctx.scale(this.dpr, this.dpr);

    this.drawTimelineBackgrounds(this.chartViewModel, this.chartStyle);
    this.drawOutsideLogPeriod(
      this.chartViewModel,
      this.chartStyle,
      leftEdgeTime,
      pixelsPerNs,
    );
    this.drawRulers(this.rulerViewModel, this.rulerStyle, pixelsPerNs);
    this.drawHorizontalLines(this.chartViewModel, this.chartStyle);
  }

  /**
   * Update the internal state with the new view models and styles.
   *
   * @param rulerViewModel - The view model for the ruler.
   * @param chartViewModel - The view model for the chart.
   * @param rulerStyle - The style configuration for the ruler.
   * @param chartStyle - The style configuration for the chart.
   * @param timelineHighlights - The highlight configuration for timelines.
   */
  update(
    rulerViewModel: TimelineRulerViewModel,
    chartViewModel: TimelineChartViewModel,
    rulerStyle: TimelineRulerStyle,
    chartStyle: TimelineChartStyle,
    timelineHighlights: TimelineHighlight,
  ) {
    this.rulerViewModel = rulerViewModel;
    this.rulerStyle = rulerStyle;
    this.chartViewModel = chartViewModel;
    this.chartStyle = chartStyle;
    this.timelineHighlights = timelineHighlights;
  }

  private drawRulers(
    viewModel: TimelineRulerViewModel,
    style: TimelineRulerStyle,
    pixelsPerMs: number,
  ) {
    let currentX = 0;
    const windowWidth = viewModel.tickTimeMS * pixelsPerMs;
    for (const tick of viewModel.ticks) {
      const t = style.rulerThicknessByImportance[tick.leftEdgeTimeImportance];
      this.ctx.lineWidth = t;
      this.ctx.strokeStyle = RendererConvertUtil.hdrColorToCSSColor(
        style.rulerColor,
      );
      this.ctx.beginPath();
      this.ctx.moveTo(currentX, 0);
      this.ctx.lineTo(currentX, this.height);
      this.ctx.stroke();
      currentX += windowWidth;
    }
  }

  private drawHorizontalLines(
    viewModel: TimelineChartViewModel,
    style: TimelineChartStyle,
  ) {
    let currentY = 0;
    for (const timeline of viewModel.timelinesInDrawArea) {
      const timelineType =
        viewModel.styleStore?.getTimelineType(timeline.type.id) ??
        timeline.type;
      const rowHeight = timelineType.height * BASE_ROW_HEIGHT;
      this.ctx.lineWidth = Math.min(
        HORIZONTAL_BORDER_THICKNESS_RANGE[1],
        Math.max(
          HORIZONTAL_BORDER_THICKNESS_RANGE[0],
          BASE_HORIZONTAL_BORDER_THICKNESS * timelineType.height,
        ),
      );
      this.ctx.strokeStyle = RendererConvertUtil.hdrColorToCSSColor(
        style.horizontalLineColor,
      );
      this.ctx.beginPath();
      this.ctx.moveTo(0, currentY);
      this.ctx.lineTo(this.width, currentY);
      this.ctx.stroke();
      currentY += rowHeight;
    }
  }

  private drawOutsideLogPeriod(
    viewModel: TimelineChartViewModel,
    style: TimelineChartStyle,
    leftEdgeTime: number,
    pixelsPerMs: number,
  ) {
    this.ctx.fillStyle = RendererConvertUtil.hdrColorToCSSColor(
      style.outsideOfLogPeriodColor,
    );
    if (leftEdgeTime < viewModel.logBeginTime) {
      this.ctx.fillRect(
        0,
        0,
        (viewModel.logBeginTime - leftEdgeTime) * pixelsPerMs,
        this.height,
      );
    }
    const rightEdgeX = (viewModel.logEndTime - leftEdgeTime) * pixelsPerMs;
    if (rightEdgeX < this.width) {
      this.ctx.fillRect(rightEdgeX, 0, this.width - rightEdgeX, this.height);
    }
  }

  private drawTimelineBackgrounds(
    viewModel: TimelineChartViewModel,
    style: TimelineChartStyle,
  ) {
    if (this.timelineHighlights === null) {
      return;
    }
    let currentY = 0;
    for (const timeline of viewModel.timelinesInDrawArea) {
      const timelineType =
        viewModel.styleStore?.getTimelineType(timeline.type.id) ??
        timeline.type;
      currentY += timelineType.height * BASE_ROW_HEIGHT;
    }
    for (let i = viewModel.timelinesInDrawArea.length - 1; i >= 0; i--) {
      const timeline = viewModel.timelinesInDrawArea[i];
      const timelineType =
        viewModel.styleStore?.getTimelineType(timeline.type.id) ??
        timeline.type;
      const rowHeight = timelineType.height * BASE_ROW_HEIGHT;
      currentY -= rowHeight;

      const isNextTimelineChild =
        i + 1 < viewModel.timelinesInDrawArea.length &&
        viewModel.timelinesInDrawArea[i + 1].layer > timeline.layer;
      const highlight = this.timelineHighlights[timeline.id];
      if (!isNextTimelineChild) {
        this.ctx.shadowColor = 'transparent';
      } else {
        this.ctx.shadowColor = 'rgba(0,0,0,0.4)';
        this.ctx.shadowOffsetY = 2;
        this.ctx.shadowOffsetX = 0;
        this.ctx.shadowBlur = 2;
      }
      // Draw the white rect at first to drop shadow not to show the entire shadow by the transparent color given by background color.
      this.ctx.fillStyle = 'white';
      this.ctx.fillRect(0, currentY, this.width, rowHeight);
      this.ctx.shadowColor = 'transparent';

      const bg = timelineType.backgroundColor;
      this.ctx.fillStyle = RendererConvertUtil.hdrColorToCSSColor([
        bg.r,
        bg.g,
        bg.b,
        bg.a,
      ]);
      this.ctx.fillRect(0, currentY, this.width, rowHeight);

      if (highlight) {
        this.ctx.save();
        this.ctx.beginPath();
        this.ctx.rect(0, currentY, this.width, rowHeight);
        this.ctx.clip();

        const color = RendererConvertUtil.hdrColorToCSSColor(
          style.timelineTintColorByHighlightType[highlight],
        );
        // 2. Draw a solid inner border
        const edgeWidth = 3;
        this.ctx.lineWidth = edgeWidth * 2;
        this.ctx.strokeStyle = color;
        this.ctx.strokeRect(0, currentY, this.width, rowHeight);

        this.ctx.restore();
      }
    }
    this.ctx.shadowColor = 'transparent';
  }
}
