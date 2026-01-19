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
  logTypeColors,
  logTypes,
  severities,
  severityColors,
} from 'src/app/generated';
import { WebGLUtil } from './glutil';
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { TimelineRendererSharedResource } from './timeline-shared-resource';
import { IDisposableRenderer, TimelineRect } from './timeline-renderer';
import { TimelineChartItemHighlight } from '../interaction-model';
import { TimelineChartStyle } from '../style-model';
import { RendererConvertUtil } from './convertutil';

/**
 * Renders timeline events (points in time) using WebGL.
 * Manages Vertex Array Objects (VAOs) and Vertex Buffer Objects (VBOs) for efficient instanced rendering.
 */
export class TimelineEventsRenderer implements IDisposableRenderer {
  public eventsVAO!: WebGLVertexArrayObject;

  private timeVBO!: WebGLBuffer;
  private intStaticMetaVBO!: WebGLBuffer;
  private intDynamicMetaVBO!: WebGLBuffer;
  private intDynamicMetaVBOSource!: Uint32Array;

  constructor(
    private timeline: ResourceTimeline,
    private eventSharedResources: TimelineEventsSharedResources,
    private timelineSharedResources: TimelineRendererSharedResource,
  ) {}

  /**
   * Sets up the WebGL resources (VAO, VBOs) for rendering events.
   * Calculates and buffers static data (time, type, severity) to the GPU.
   *
   * @param gl The WebGL2 rendering context.
   */
  setup(gl: WebGL2RenderingContext) {
    const timeVBOSource = new Uint32Array(this.timeline.events.length * 2);
    for (let i = 0; i < this.timeline.events.length; i++) {
      const event = this.timeline.events[i];
      const start = RendererConvertUtil.splitTimeToSecondsAndNanoSeconds(
        event.ts,
      );
      timeVBOSource[i * 2] = start[0];
      timeVBOSource[i * 2 + 1] = start[1];
    }
    this.timeVBO = gl.createBuffer();
    if (this.timeVBO === null) {
      throw new Error('Failed to create time VBO');
    }
    gl.bindBuffer(gl.ARRAY_BUFFER, this.timeVBO);
    gl.bufferData(gl.ARRAY_BUFFER, timeVBOSource, gl.STATIC_DRAW);

    const intStaticMetaVBOSource = new Uint32Array(
      this.timeline.events.length * 4,
    );
    for (let i = 0; i < this.timeline.events.length; i++) {
      const event = this.timeline.events[i];
      intStaticMetaVBOSource[i * 4] = i;
      intStaticMetaVBOSource[i * 4 + 1] = event.logType;
      intStaticMetaVBOSource[i * 4 + 2] = event.logSeverity;
      intStaticMetaVBOSource[i * 4 + 3] = 0;
    }
    this.intStaticMetaVBO = gl.createBuffer();
    if (this.intStaticMetaVBO === null) {
      throw new Error('Failed to create intStaticMeta VBO');
    }
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intStaticMetaVBO);
    gl.bufferData(gl.ARRAY_BUFFER, intStaticMetaVBOSource, gl.STATIC_DRAW);

    this.intDynamicMetaVBOSource = new Uint32Array(
      this.timeline.events.length * 4,
    );
    this.intDynamicMetaVBO = gl.createBuffer();
    if (this.intDynamicMetaVBO === null) {
      throw new Error('Failed to create intDynamicMeta VBO');
    }
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intDynamicMetaVBO);
    gl.bufferData(
      gl.ARRAY_BUFFER,
      this.intDynamicMetaVBOSource,
      gl.DYNAMIC_DRAW,
    );

    this.eventsVAO = gl.createVertexArray();
    if (this.eventsVAO === null) {
      throw new Error('Failed to create events VAO');
    }
    gl.bindVertexArray(this.eventsVAO);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.timeVBO);
    gl.vertexAttribIPointer(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_TIME,
      2,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_TIME,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_TIME,
    );
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intStaticMetaVBO);
    gl.vertexAttribIPointer(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
      4,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
    );
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intDynamicMetaVBO);
    gl.vertexAttribIPointer(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
      4,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineEventsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
    );
    gl.bindVertexArray(null);
    gl.bindBuffer(gl.ARRAY_BUFFER, null);
  }

  /**
   * Updates the dynamic buffer with highlight/selection status for each event.
   * This is called when the user interacts with the timeline.
   *
   * @param gl The WebGL rendering context.
   * @param logElementHighlights Map of log indices to their highlight state.
   * @param activeLogsIndices Set of log indices that are currently active(not filtered out).
   */
  updateDynamicBuffer(
    gl: WebGLRenderingContext,
    logElementHighlights: TimelineChartItemHighlight,
    activeLogsIndices: Set<number>,
  ) {
    for (let i = 0; i < this.timeline.events.length; i++) {
      const selectionStatus =
        logElementHighlights[this.timeline.events[i].logIndex] ?? 0;
      const filterStatus = activeLogsIndices.has(
        this.timeline.events[i].logIndex,
      )
        ? 1
        : 0;
      this.intDynamicMetaVBOSource[i * 4 + 0] = selectionStatus;
      this.intDynamicMetaVBOSource[i * 4 + 1] = filterStatus;
    }
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intDynamicMetaVBO);
    gl.bufferData(
      gl.ARRAY_BUFFER,
      this.intDynamicMetaVBOSource,
      gl.DYNAMIC_DRAW,
    );
    gl.bindBuffer(gl.ARRAY_BUFFER, null);
  }

  /**
   * Renders the events to the specified rectangle.
   *
   * @param gl The WebGL2 rendering context.
   * @param rect The screen rectangle where the events should be drawn.
   */
  renderColor(gl: WebGL2RenderingContext, rect: TimelineRect) {
    if (!this.eventSharedResources.eventLayerStylesUBO) {
      return;
    }
    gl.enable(gl.DEPTH_TEST);
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA); // transparent canvas expects premultiplied alpha result

    this.renderInstances(
      gl,
      this.eventSharedResources.eventsColorProgram,
      rect,
    );
  }

  /**
   * Renders the events for hit testing (off-screen).
   * Generates a color-coded map where each pixel corresponds to an event index.
   *
   * @param gl The WebGL2 rendering context.
   * @param rect The screen rectangle where the hit test is being performed.
   */
  renderHittest(gl: WebGL2RenderingContext, rect: TimelineRect) {
    if (!this.eventSharedResources.eventLayerStylesUBO) {
      return;
    }
    gl.disable(gl.BLEND);
    gl.enable(gl.DEPTH_TEST);

    this.renderInstances(
      gl,
      this.eventSharedResources.eventsHittestProgram,
      rect,
    );
  }

  private renderInstances(
    gl: WebGL2RenderingContext,
    program: WebGLProgram,
    rect: TimelineRect,
  ) {
    gl.viewport(
      0,
      rect.offsetY * rect.dpr,
      rect.width * rect.dpr,
      rect.height * rect.dpr,
    );
    gl.useProgram(program);
    gl.bindVertexArray(this.eventsVAO);

    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineEventsSharedResources.UBO_BINDING_VIEW_STATE,
      this.timelineSharedResources.uboViewState,
    );
    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineEventsSharedResources.UBO_BINDING_EVENT_STYLES,
      this.eventSharedResources.eventStylesUBO,
    );
    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineEventsSharedResources.UBO_BINDING_EVENT_LAYER_STYLES,
      this.eventSharedResources.eventLayerStylesUBO[this.timeline.layer],
    );

    gl.drawArraysInstanced(
      gl.TRIANGLE_STRIP,
      0,
      4,
      this.timeline.events.length,
    );

    gl.bindVertexArray(null);
    gl.useProgram(null);
  }

  /**
   * Frees WebGL resources associated with this renderer.
   *
   * @param gl The WebGL2 rendering context.
   */
  dispose(gl: WebGL2RenderingContext) {
    gl.deleteVertexArray(this.eventsVAO);
    gl.deleteBuffer(this.timeVBO);
    gl.deleteBuffer(this.intStaticMetaVBO);
    gl.deleteBuffer(this.intDynamicMetaVBO);
  }
}

/**
 * Manages shared WebGL resources for rendering timeline events.
 * This includes shader programs and style buffers.
 */
export class TimelineEventsSharedResources {
  public static readonly MAX_LOG_TYPE_COUNT = 128;
  public static readonly MAX_SEVERITY_COUNT = 8;

  public static readonly UBO_BINDING_VIEW_STATE = 0;
  public static readonly UBO_BINDING_EVENT_STYLES = 1;
  public static readonly UBO_BINDING_EVENT_LAYER_STYLES = 2;

  public static readonly VBO_LAYOUT_LOCATION_TIME = 0;
  public static readonly VBO_LAYOUT_LOCATION_INT_STATIC_META = 1;
  public static readonly VBO_LAYOUT_LOCATION_INT_DYNAMIC_META = 2;

  public eventsColorProgram!: WebGLProgram;

  public eventsHittestProgram!: WebGLProgram;

  public eventStylesUBO!: WebGLBuffer;

  /**
   * UBOs storeing styles to render events for each layer.
   * Styles bound to its layer are stored in eventStylesUBO.
   */
  public eventLayerStylesUBO!: { [key in TimelineLayer]: WebGLBuffer };

  private chartStyle!: TimelineChartStyle;

  private styleUpdated = false;

  /**
   * Initializes shaders and buffers for event rendering.
   *
   * @param gl The WebGL2 rendering context.
   */
  async setup(gl: WebGL2RenderingContext) {
    this.eventsColorProgram = await WebGLUtil.compileAndLinkShaders(
      gl,
      'assets/event-v2.vertex.glsl',
      'assets/event-v2.color-fragment.glsl',
      {
        '#include "v2.shared.glsl"': 'assets/v2.shared.glsl',
        '#include "event-v2.shared.glsl"': 'assets/event-v2.shared.glsl',
      },
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsColorProgram,
      'ViewState',
      TimelineEventsSharedResources.UBO_BINDING_VIEW_STATE,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsColorProgram,
      'EventStyles',
      TimelineEventsSharedResources.UBO_BINDING_EVENT_STYLES,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsColorProgram,
      'EventLayerStyles',
      TimelineEventsSharedResources.UBO_BINDING_EVENT_LAYER_STYLES,
    );

    this.eventsHittestProgram = await WebGLUtil.compileAndLinkShaders(
      gl,
      'assets/event-v2.vertex.glsl',
      'assets/event-v2.hittest-fragment.glsl',
      {
        '#include "v2.shared.glsl"': 'assets/v2.shared.glsl',
        '#include "event-v2.shared.glsl"': 'assets/event-v2.shared.glsl',
      },
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsHittestProgram,
      'ViewState',
      TimelineEventsSharedResources.UBO_BINDING_VIEW_STATE,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsHittestProgram,
      'EventStyles',
      TimelineEventsSharedResources.UBO_BINDING_EVENT_STYLES,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.eventsHittestProgram,
      'EventLayerStyles',
      TimelineEventsSharedResources.UBO_BINDING_EVENT_LAYER_STYLES,
    );

    if (logTypes.length > TimelineEventsSharedResources.MAX_LOG_TYPE_COUNT) {
      throw new Error(
        'Too many log types: Consider increassing the constant variables defined in the shader.',
      );
    }
    if (severities.length > TimelineEventsSharedResources.MAX_SEVERITY_COUNT) {
      throw new Error(
        'Too many severities: Consider increassing the constant variables defined in the shader.',
      );
    }

    const uboSource = new Float32Array(
      TimelineEventsSharedResources.MAX_LOG_TYPE_COUNT * 4 +
        TimelineEventsSharedResources.MAX_SEVERITY_COUNT * 4,
    );
    for (let i = 0; i < logTypes.length; i++) {
      const color = logTypeColors[logTypes[i]];
      uboSource[i * 4] = color[0];
      uboSource[i * 4 + 1] = color[1];
      uboSource[i * 4 + 2] = color[2];
      uboSource[i * 4 + 3] = 0;
    }
    const baseOffset = TimelineEventsSharedResources.MAX_LOG_TYPE_COUNT * 4;
    for (let i = 0; i < severities.length; i++) {
      const color = severityColors[severities[i]];
      uboSource[baseOffset + i * 4] = color[0];
      uboSource[baseOffset + i * 4 + 1] = color[1];
      uboSource[baseOffset + i * 4 + 2] = color[2];
      uboSource[baseOffset + i * 4 + 3] = 0;
    }

    this.eventStylesUBO = gl.createBuffer()!;
    if (this.eventStylesUBO === null) {
      throw new Error('Failed to create eventStyles UBO');
    }
    gl.bindBuffer(gl.UNIFORM_BUFFER, this.eventStylesUBO);
    gl.bufferData(gl.UNIFORM_BUFFER, uboSource, gl.STATIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);
  }

  /**
   * Updates the style buffers if the chart style has changed.
   *
   * @param gl The WebGL2 rendering context.
   */
  beforeRender(gl: WebGL2RenderingContext) {
    if (this.styleUpdated) {
      this.eventLayerStylesUBO = {
        [TimelineLayer.APIVersion]: this.createStyleUBOForLayer(
          gl,
          TimelineLayer.APIVersion,
        ),
        [TimelineLayer.Kind]: this.createStyleUBOForLayer(
          gl,
          TimelineLayer.Kind,
        ),
        [TimelineLayer.Namespace]: this.createStyleUBOForLayer(
          gl,
          TimelineLayer.Namespace,
        ),
        [TimelineLayer.Name]: this.createStyleUBOForLayer(
          gl,
          TimelineLayer.Name,
        ),
        [TimelineLayer.Subresource]: this.createStyleUBOForLayer(
          gl,
          TimelineLayer.Subresource,
        ),
      };
      this.styleUpdated = false;
    }
  }

  /**
   * Updates the chart style to be used for rendering.
   * This triggers a style update on the next frame.
   *
   * @param chartStyle The new timeline chart style.
   */
  updateChartStyle(chartStyle: TimelineChartStyle) {
    this.chartStyle = chartStyle;
    this.styleUpdated = true;
  }

  private createStyleUBOForLayer(
    gl: WebGL2RenderingContext,
    layer: TimelineLayer,
  ): WebGLBuffer {
    const ubo = gl.createBuffer();
    if (ubo === null) {
      throw new Error('Failed to create layer style UBO');
    }
    gl.bindBuffer(gl.UNIFORM_BUFFER, ubo);
    const timelineHeight = this.chartStyle.heightsByLayer[layer];
    const eventStyle = this.chartStyle.eventStylesByLayer[layer];
    const STD140_PADDING_FLOAT = 0;
    const hoverBorderColorInVec3 = this.chartStyle.highlightBorderColor;
    const selectionBorderColorInVec3 = this.chartStyle.selectionBorderColor;
    const bufferSource = new Float32Array([
      timelineHeight,
      eventStyle.verticalPaddingInPx,
      eventStyle.severityColorRatio,
      eventStyle.borderThickness,
      eventStyle.borderAntialias,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      hoverBorderColorInVec3[0],
      hoverBorderColorInVec3[1],
      hoverBorderColorInVec3[2],
      eventStyle.hoverBorderThickness,
      selectionBorderColorInVec3[0],
      selectionBorderColorInVec3[1],
      selectionBorderColorInVec3[2],
      eventStyle.selectionBorderThickness,
    ]);
    gl.bufferData(gl.UNIFORM_BUFFER, bufferSource, gl.STATIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);
    return ubo;
  }
}
