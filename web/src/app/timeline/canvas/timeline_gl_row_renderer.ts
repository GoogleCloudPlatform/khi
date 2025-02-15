/**
 * Copyright 2024 Google LLC
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

import { GLRect, GLResource, TIMELINE_ITEM_HEIGHTS } from './types';
import { SharedGLResources } from './shared_gl_resource';
import {
  revisionStateToIndex,
  revisionStates,
  severities,
} from 'src/app/generated';
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { TimelineGLResourceManager } from './timeline_gl_resource_manager';
import { Subject, takeUntil } from 'rxjs';

/**
 * A enum type representing the selection status of each revision/event and this value is directly passed to shader side.
 */
enum SelectionStatusForShader {
  FilteredOut = 0,
  Default = 1,
  Highlighted = 2,
  Selected = 3,
}

/**
 * TimelineRowWebGLRenderer draws a single row of timeline.
 */
export class TimelineRowWebGLRenderer extends GLResource {
  private revisionTimeBuffer!: WebGLBuffer;

  private revisionTimeBufferSource!: Float32Array;

  /**
   * Float typed revision data.
   * x = revision state
   */
  private revisionMetaBuffer!: WebGLBuffer;

  private revisionMetaBufferSource!: Float32Array;

  /**
   * Int typed metadata per revision instance.
   * x = revisonIndex, y = interaction status(SelectionStatusForShader)
   */
  private revisionIntMetaBuffer!: WebGLBuffer;

  private revisionIntMetaBufferSource!: Int32Array;

  private eventBuffer!: WebGLBuffer;

  private eventBufferSource!: Float32Array;

  /**
   * Int typed metadata per event instance.
   * x = interaction status(0:none,1:highlight,2:selected)
   */
  private eventIntMetaBuffer!: WebGLBuffer;

  private eventIntMetaBufferSource!: Int32Array;

  private initialized = false;

  private isGLResourceLoaded = false;

  private destroyed = new Subject<void>();

  constructor(
    gl: WebGL2RenderingContext,
    private resourceManager: TimelineGLResourceManager,
    private sharedResources: SharedGLResources,
    private timeline: ResourceTimeline,
    private leftMostTime: number,
  ) {
    super(gl);
    this.initializeNonGLResources();
    this.resourceManager
      .onUnload(this.timeline.resourcePath)
      .pipe(takeUntil(this.destroyed))
      .subscribe(() => {
        this.unloadGLResources();
      });
  }

  /**
   * Construct all resources not related to WebGL.
   * All resources constucted on this method will be kept on memory while this inspection data is opened.
   */
  private initializeNonGLResources() {
    if (this.initialized) return;
    const selectionStatus = [];
    for (let i = 0; i < this.timeline.events.length; i++) {
      selectionStatus.push(0);
    }
    this.eventIntMetaBufferSource = new Int32Array(selectionStatus);

    const revisionIntMeta = [];
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      revisionIntMeta.push(i, 0);
    }
    this.revisionIntMetaBufferSource = new Int32Array(revisionIntMeta);

    const timePoints = [];
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      const revision = this.timeline.revisions[i];
      // To avoid the precision problem of floating values in timestamp, use the time relative to the left most timestamp of the log.
      timePoints.push(
        revision.startAt - this.leftMostTime,
        revision.endAt - this.leftMostTime,
      );
    }
    this.revisionTimeBufferSource = new Float32Array(timePoints);

    const metaPoints: number[] = [];
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      metaPoints.push(
        revisionStateToIndex[
          this.timeline.revisions[i].revisionStateCssSelector
        ],
        0,
      );
    }
    this.revisionMetaBufferSource = new Float32Array(metaPoints);

    const events = [];
    for (let i = 0; i < this.timeline.events.length; i++) {
      const event = this.timeline.events[i];
      events.push(
        event.ts - this.leftMostTime, // event.x = time offset from left
        event.logType, // event.y = log type
        event.logSeverity / severities.length + 0.001, // event.z = log severity index / log severity type count + small pad not to sample on the border
      );
    }
    this.eventBufferSource = new Float32Array(events);

    this.initialized = true;
  }

  /**
   * Load buffers needed to render on WebGL.
   * This may keep data on GPU VRAM but it can be released when user opens a big inspection data.
   *
   */
  public loadGLResources() {
    this.resourceManager.touch(this.timeline.resourcePath);
    if (this.isGLResourceLoaded) return;

    // Initialize revision time buffer containing series of times on this timeline.
    this.revisionTimeBuffer = this.glMust(this.gl.createBuffer());

    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.revisionTimeBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.revisionTimeBufferSource,
      this.gl.STATIC_DRAW,
    );

    // Initialize revision meta buffer containing information needed to render revision rectangles in the style.
    this.revisionMetaBuffer = this.glMust(this.gl.createBuffer());

    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.revisionMetaBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.revisionMetaBufferSource,
      this.gl.STATIC_DRAW,
    );

    // Initialize revision int meta buffer conaining indices of revisions.
    this.revisionIntMetaBuffer = this.glMust(this.gl.createBuffer());
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.revisionIntMetaBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.revisionIntMetaBufferSource,
      this.gl.DYNAMIC_DRAW,
    );

    this.eventBuffer = this.glMust(this.gl.createBuffer());

    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.eventBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.eventBufferSource,
      this.gl.STATIC_DRAW,
    );

    this.eventIntMetaBuffer = this.glMust(this.gl.createBuffer());
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.eventIntMetaBuffer);
    this.gl.bufferData(
      this.gl.ARRAY_BUFFER,
      this.eventIntMetaBufferSource,
      this.gl.DYNAMIC_DRAW,
    );

    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, null);

    this.isGLResourceLoaded = true;
  }

  /**
   * Unload buffers on WebGL.
   */
  public unloadGLResources() {
    if (!this.isGLResourceLoaded) return;
    this.gl.deleteBuffer(this.revisionTimeBuffer);
    this.gl.deleteBuffer(this.revisionMetaBuffer);
    this.gl.deleteBuffer(this.revisionIntMetaBuffer);
    this.gl.deleteBuffer(this.eventBuffer);
    this.gl.deleteBuffer(this.eventIntMetaBuffer);

    this.isGLResourceLoaded = false;
  }

  /**
   * Update buffers in response to the change of (selected|highlighted) log indices.
   */
  public updateInteractiveBuffer(
    selectedLogIndex: number,
    highlightedLogIndices: Set<number>,
    filteredLogIndices: Set<number>,
  ) {
    this.ignoreGLContextLostException(() => {
      for (const [index, revision] of this.timeline.revisions.entries()) {
        let status = SelectionStatusForShader.Default;
        if (filteredLogIndices.has(revision.logIndex)) {
          status = SelectionStatusForShader.FilteredOut;
        } else if (revision.revisionStateCssSelector !== 'inferred') {
          // When the timeline is `inferred`, there are no actual related log. shouldn't be highlighted/selected.
          if (
            selectedLogIndex !== -1 &&
            revision.logIndex === selectedLogIndex
          ) {
            status = SelectionStatusForShader.Selected;
          } else if (highlightedLogIndices.has(revision.logIndex)) {
            status = SelectionStatusForShader.Highlighted;
          }
        }
        this.revisionIntMetaBufferSource[index * 2 + 1] = status;
      }
      this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.revisionIntMetaBuffer);
      this.gl.bufferData(
        this.gl.ARRAY_BUFFER,
        this.revisionIntMetaBufferSource,
        this.gl.DYNAMIC_DRAW,
      );

      for (const [index, event] of this.timeline.events.entries()) {
        let status = SelectionStatusForShader.Default;
        if (filteredLogIndices.has(event.logIndex)) {
          status = SelectionStatusForShader.FilteredOut;
        } else {
          if (selectedLogIndex !== -1 && event.logIndex === selectedLogIndex) {
            status = SelectionStatusForShader.Selected;
          } else if (highlightedLogIndices.has(event.logIndex)) {
            status = SelectionStatusForShader.Highlighted;
          }
        }
        this.eventIntMetaBufferSource[index] = status;
      }
      this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.eventIntMetaBuffer);
      this.gl.bufferData(
        this.gl.ARRAY_BUFFER,
        this.eventIntMetaBufferSource,
        this.gl.DYNAMIC_DRAW,
      );
    });
  }

  /**
   * Render the timeline on the specified region.
   */
  public render(
    canvasPixelRatio: number,
    rect: GLRect,
    timelineSelected: boolean,
    timelineHighlighted: boolean,
    timelineHighightedByParent: boolean,
  ) {
    this.ignoreGLContextLostException(() => {
      const gl = this.gl;
      const timelineHeight =
        TIMELINE_ITEM_HEIGHTS[this.timeline.layer] * canvasPixelRatio;

      // clear buffer
      gl.enable(gl.SCISSOR_TEST);
      gl.scissor(rect.left, rect.bottom, rect.width, rect.height);
      gl.clearColor(
        ...this.getBackgroundFillColor(
          this.timeline.layer,
          timelineSelected,
          timelineHighlighted,
          timelineHighightedByParent,
        ),
      );
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
      gl.viewport(rect.left, rect.bottom, rect.width, rect.height);
      gl.enable(gl.BLEND);
      gl.disable(gl.DEPTH_TEST);
      gl.depthMask(false);

      // Draw border lines
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
      gl.useProgram(this.sharedResources.borderShaderProgram);
      gl.bindVertexArray(this.sharedResources.vaoRectangle);
      gl.bindBufferBase(
        gl.UNIFORM_BUFFER,
        0,
        this.sharedResources.uboViewState,
      );
      gl.uniform1f(
        gl.getUniformLocation(
          this.sharedResources.borderShaderProgram,
          'timelineHeight',
        ),
        timelineHeight,
      );
      gl.drawElements(gl.TRIANGLES, 6, gl.UNSIGNED_BYTE, 0);

      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);

      // Draw revision rectangles
      const TIME_BUFFER_LOCATION = 1;
      const META_BUFFER_LOCATION = 2;
      const REVISION_INDEX_BUFFER_LOCATION = 3;
      gl.useProgram(this.sharedResources.revisionShaderProgram);
      gl.bindVertexArray(this.sharedResources.vaoRectangle);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.revisionTimeBuffer);
      gl.enableVertexAttribArray(TIME_BUFFER_LOCATION);
      gl.vertexAttribPointer(TIME_BUFFER_LOCATION, 2, gl.FLOAT, false, 0, 0);
      gl.vertexAttribDivisor(TIME_BUFFER_LOCATION, 1);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.revisionMetaBuffer);
      gl.enableVertexAttribArray(META_BUFFER_LOCATION);
      gl.vertexAttribPointer(META_BUFFER_LOCATION, 2, gl.FLOAT, false, 0, 0);
      gl.vertexAttribDivisor(META_BUFFER_LOCATION, 1);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.revisionIntMetaBuffer);
      gl.enableVertexAttribArray(REVISION_INDEX_BUFFER_LOCATION);
      gl.vertexAttribIPointer(REVISION_INDEX_BUFFER_LOCATION, 2, gl.INT, 0, 0);
      gl.vertexAttribDivisor(REVISION_INDEX_BUFFER_LOCATION, 1);

      gl.uniform1i(
        gl.getUniformLocation(
          this.sharedResources.revisionShaderProgram,
          'numberTexture',
        ),
        0,
      );
      gl.uniform1i(
        gl.getUniformLocation(
          this.sharedResources.revisionShaderProgram,
          'revisionColorPalette',
        ),
        2,
      );
      gl.uniform1f(
        gl.getUniformLocation(
          this.sharedResources.revisionShaderProgram,
          'revisionStateCount',
        ),
        revisionStates.length,
      );
      gl.uniform1f(
        gl.getUniformLocation(
          this.sharedResources.revisionShaderProgram,
          'timelineHeight',
        ),
        timelineHeight,
      );
      gl.uniform1f(
        gl.getUniformLocation(
          this.sharedResources.revisionShaderProgram,
          'devicePixelRatio',
        ),
        window.devicePixelRatio,
      );

      gl.bindBufferBase(
        gl.UNIFORM_BUFFER,
        0,
        this.sharedResources.uboViewState,
      );

      gl.drawElementsInstanced(
        gl.TRIANGLES,
        6,
        gl.UNSIGNED_BYTE,
        0,
        this.timeline.revisions.length,
      );
      gl.bindVertexArray(null);

      // Draw event rectangles
      gl.enable(gl.DEPTH_TEST);
      gl.depthMask(true);
      const EVENT_BUFFER_LOCATION = 1;
      const EVENT_META_BUFFER_LOCATION = 2;
      gl.useProgram(this.sharedResources.eventShaderProgram);
      gl.bindVertexArray(this.sharedResources.vaoRectangle);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.eventBuffer);
      gl.enableVertexAttribArray(EVENT_BUFFER_LOCATION);
      gl.vertexAttribPointer(EVENT_BUFFER_LOCATION, 3, gl.FLOAT, false, 0, 0);
      gl.vertexAttribDivisor(EVENT_BUFFER_LOCATION, 1);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.eventIntMetaBuffer);
      gl.enableVertexAttribArray(EVENT_META_BUFFER_LOCATION);
      gl.vertexAttribIPointer(EVENT_META_BUFFER_LOCATION, 1, gl.INT, 0, 0);
      gl.vertexAttribDivisor(EVENT_META_BUFFER_LOCATION, 1);

      gl.uniform1i(
        gl.getUniformLocation(
          this.sharedResources.eventShaderProgram,
          'colorPaletteTexture',
        ),
        1,
      );
      gl.uniform1i(
        gl.getUniformLocation(
          this.sharedResources.eventShaderProgram,
          'logSeverityColorPaletteTexture',
        ),
        3,
      );
      gl.uniform1f(
        gl.getUniformLocation(
          this.sharedResources.eventShaderProgram,
          'timelineHeight',
        ),
        timelineHeight,
      );

      gl.bindBufferBase(
        gl.UNIFORM_BUFFER,
        0,
        this.sharedResources.uboViewState,
      );

      gl.drawElementsInstanced(
        gl.TRIANGLES,
        6,
        gl.UNSIGNED_BYTE,
        0,
        this.timeline.events.length,
      );
      gl.bindVertexArray(null);
    });
  }

  /**
   * Returns color of background from given timeline status.
   * @param layer the layer of timeline
   * @param selected if the timeline is selected
   * @param highlighted if the timeline is directly highlighted by user interaction
   * @param highlightedByParent if the timeline is inheriting the highlight status from the parent timeline
   * @returns 4 numbers array representing the color
   */
  private getBackgroundFillColor(
    layer: TimelineLayer,
    selected: boolean,
    highlighted: boolean,
    highlightedByParent: boolean,
  ): [number, number, number, number] {
    let backgroundColor: number[] = [0, 0, 0];
    let backgroundAlpha = 0;
    if (layer === TimelineLayer.Kind) {
      backgroundColor = [63 / 255, 81 / 255, 181 / 255];
      backgroundAlpha = 0.9;
    } else if (layer === TimelineLayer.Namespace) {
      backgroundColor = [100 / 255, 100 / 255, 100 / 255];
      backgroundAlpha = 0.9;
    } else if (layer === TimelineLayer.Name) {
      backgroundColor = [0, 0, 0];
      backgroundAlpha = 0.2;
    }
    const tintBase = [0.25, 0.73, 0.55];
    if (selected) {
      backgroundColor = this.tintColor3(backgroundColor, tintBase, 0.45, 0.5);
      backgroundAlpha = 0.6;
    } else if (highlighted) {
      backgroundColor = this.tintColor3(backgroundColor, tintBase, 0.55, 0.5);
      backgroundAlpha = 0.6;
    } else if (highlightedByParent) {
      backgroundColor = this.tintColor3(backgroundColor, tintBase, 0.7, 0.5);
      backgroundAlpha = 0.5;
    }
    return [
      backgroundColor[0],
      backgroundColor[1],
      backgroundColor[2],
      backgroundAlpha,
    ];
  }

  public override dispose() {
    this.unloadGLResources();
    this.destroyed.next();
  }

  private tintColor3(
    color: number[],
    tintColor: number[],
    power: number,
    ratio: number,
  ): number[] {
    const result = [0, 0, 0];
    for (let i = 0; i < 3; i++) {
      result[i] = color[i] * ratio + (1 - ratio) * tintColor[i] * power;
    }
    return result;
  }
}
