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

import { TimelineRulerStyle, generateDefaultRulerStyle } from '../style-model';
import { TimelineRulerViewModel } from '../timeline-ruler.viewmodel';
import { RendererConvertUtil } from './convertutil';

/**
 * Renders the timeline ruler including histograms and time ticks using Canvas 2D API.
 * This class handles the actual drawing commands based on the provided view model.
 */
export class TimelineRulerRenderer {
  private width = 0;
  private height = 0;
  private style: TimelineRulerStyle;

  /**
   * Creates a new instance of TimelineRulerRenderer.
   * @param ctx The 2D rendering context of the canvas.
   */
  constructor(private ctx: CanvasRenderingContext2D) {
    this.style = generateDefaultRulerStyle();
  }

  /**
   * Resizes the renderer and adjusts the canvas context scaling.
   *
   * @param width The logical width of the canvas in pixels.
   * @param height The logical height of the canvas in pixels.
   * @param dpr The device pixel ratio (e.g. window.devicePixelRatio).
   */
  resize(width: number, height: number, dpr: number) {
    this.width = width;
    this.height = height;
    this.ctx.setTransform(1, 0, 0, 1, 0, 0);
    this.ctx.scale(dpr, dpr);
  }

  /**
   * Clears the canvas and renders the ruler components: histogram, foreground overlay, and ruler lines.
   *
   * @param viewModel The view model containing data to render (histogram buckets, ticks).
   * @param leftEdgeTimeMS The time at the left edge of the viewport in milliseconds.
   * @param pixelsPerMs The current scale in pixels per millisecond.
   */
  render(
    viewModel: TimelineRulerViewModel,
    leftEdgeTimeMS: number,
    pixelsPerMs: number,
  ) {
    this.ctx.clearRect(0, 0, this.width, this.height);
    this.drawHeaderHistogram(viewModel, leftEdgeTimeMS, pixelsPerMs);
    this.drawRulerLines(viewModel, pixelsPerMs);
  }

  /**
   * Draws the severity histogram background.
   */
  private drawHeaderHistogram(
    viewModel: TimelineRulerViewModel,
    leftEdgeTime: number,
    pixelsPerMs: number,
  ) {
    const windowWidth = viewModel.histogramBucketTimeMS * pixelsPerMs;
    let currentX =
      (viewModel.histogramBeginTimeMS - leftEdgeTime) * pixelsPerMs;
    const t = this.style.histogramLineThickness / 2;
    this.ctx.lineWidth = this.style.histogramLineThickness;
    const headerHeight = this.style.headerHeightInPx;
    const histogramHeight = this.style.maxHistogramHeightInPx;

    for (const window of viewModel.histogramBuckets) {
      const barGroups = [
        { ratios: window.all, alpha: this.style.nonHighlightedAlpha },
        { ratios: window.highlighted, alpha: this.style.highlightedAlpha },
      ];
      for (const group of barGroups) {
        let currentY = 0;
        for (const severity of this.style.severitiesInDrawOrder) {
          const ratio = group.ratios[severity];
          if (ratio === 0) continue;
          this.ctx.fillStyle = RendererConvertUtil.hdrColorToCSSColorWithAlpha(
            this.style.severityColors[severity],
            group.alpha,
          );
          this.ctx.strokeStyle =
            RendererConvertUtil.hdrColorToCSSColorWithAlpha(
              this.style.severityStrokeColors[severity],
              group.alpha * 0.5,
            );
          const height = ratio * histogramHeight;
          this.ctx.fillRect(
            currentX + t,
            headerHeight - currentY - height + t,
            windowWidth - t * 2,
            height - t * 2,
          );
          this.ctx.strokeRect(
            currentX + t,
            headerHeight - currentY - height + t,
            windowWidth - t * 2,
            height - t * 2,
          );
          currentY += height;
        }
      }
      currentX += windowWidth;
    }
  }

  /**
   * Draws the vertical ruler lines (ticks) indicating time intervals.
   */
  private drawRulerLines(
    viewModel: TimelineRulerViewModel,
    pixelsPerMs: number,
  ) {
    const windowWidth = viewModel.tickTimeMS * pixelsPerMs;
    let currentX = 0;
    for (const window of viewModel.ticks) {
      const t =
        this.style.rulerThicknessByImportance[window.leftEdgeTimeImportance];
      this.ctx.lineWidth = t;
      this.ctx.strokeStyle = RendererConvertUtil.hdrColorToCSSColor(
        this.style.rulerColor,
      );
      this.ctx.beginPath();
      this.ctx.moveTo(
        currentX,
        this.style.rulerExtraHeightByImportance[window.leftEdgeTimeImportance],
      );
      this.ctx.lineTo(currentX, 0);
      this.ctx.stroke();
      currentX += windowWidth;
    }
  }
}
