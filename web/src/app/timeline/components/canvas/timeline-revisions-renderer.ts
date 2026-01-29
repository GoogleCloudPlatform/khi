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

import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { WebGLUtil } from './glutil';
import {
  revisionStatecolors,
  RevisionStateMetadata,
  revisionStates,
  revisionStateToIndex,
} from 'src/app/zzz-generated';
import { TimelineRendererSharedResource } from './timeline-shared-resource';
import { IDisposableRenderer, TimelineRect } from './timeline-renderer';
import {
  TimelineChartItemHighlight,
  TimelineChartItemHighlightType,
} from '../interaction-model';
import { TimelineChartStyle } from '../style-model';
import { RendererConvertUtil } from './convertutil';

/**
 * Renders timeline revisions (horizontal bars representing resource states) using WebGL.
 * Manages Vertex Array Objects (VAOs) and Vertex Buffer Objects (VBOs) for efficient instanced rendering.
 */
export class TimelineRevisionsRenderer implements IDisposableRenderer {
  public revisionsVAO!: WebGLVertexArrayObject;

  private timeVBO!: WebGLBuffer;
  private intStaticMetaVBO!: WebGLBuffer;
  private intDynamicMetaVBO!: WebGLBuffer;
  private intDynamicMetaVBOSource!: Uint32Array;

  constructor(
    private timeline: ResourceTimeline,
    private revisionSharedResources: TimelineRevisionsSharedResources,
    private timelineSharedResources: TimelineRendererSharedResource,
  ) {}

  /**
   * Sets up the WebGL resources (VAO, VBOs) for rendering revisions.
   * Calculates and buffers static data (time, state metadata) to the GPU.
   *
   * @param gl The WebGL2 rendering context.
   */
  setup(gl: WebGL2RenderingContext): void {
    const timeVBOSource = new Uint32Array(this.timeline.revisions.length * 4);
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      const revision = this.timeline.revisions[i];
      const start = RendererConvertUtil.splitTimeToSecondsAndNanoSeconds(
        revision.startAt,
      );
      const end = RendererConvertUtil.splitTimeToSecondsAndNanoSeconds(
        revision.endAt,
      );
      timeVBOSource[i * 4] = start[0];
      timeVBOSource[i * 4 + 1] = start[1];
      timeVBOSource[i * 4 + 2] = end[0];
      timeVBOSource[i * 4 + 3] = end[1];
    }
    this.timeVBO = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.timeVBO);
    gl.bufferData(gl.ARRAY_BUFFER, timeVBOSource, gl.STATIC_DRAW);

    const intStaticMetaVBOSource = new Uint32Array(
      this.timeline.revisions.length * 4,
    );
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      const revision = this.timeline.revisions[i];
      intStaticMetaVBOSource[i * 4] = i;
      intStaticMetaVBOSource[i * 4 + 1] =
        revisionStateToIndex[revision.revisionStateCssSelector];
      intStaticMetaVBOSource[i * 4 + 2] = revision.logIndex;
      intStaticMetaVBOSource[i * 4 + 3] = 0;
    }
    this.intStaticMetaVBO = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intStaticMetaVBO);
    gl.bufferData(gl.ARRAY_BUFFER, intStaticMetaVBOSource, gl.STATIC_DRAW);

    this.intDynamicMetaVBOSource = new Uint32Array(
      this.timeline.revisions.length * 4,
    );
    this.intDynamicMetaVBO = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intDynamicMetaVBO);
    gl.bufferData(
      gl.ARRAY_BUFFER,
      this.intDynamicMetaVBOSource,
      gl.DYNAMIC_DRAW,
    );

    this.revisionsVAO = gl.createVertexArray();
    gl.bindVertexArray(this.revisionsVAO);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.timeVBO);
    gl.vertexAttribIPointer(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_TIME,
      4,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_TIME,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_TIME,
    );
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intStaticMetaVBO);
    gl.vertexAttribIPointer(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
      4,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_STATIC_META,
    );
    gl.bindBuffer(gl.ARRAY_BUFFER, this.intDynamicMetaVBO);
    gl.vertexAttribIPointer(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
      4,
      gl.UNSIGNED_INT,
      0,
      0,
    );
    gl.vertexAttribDivisor(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
      1,
    );
    gl.enableVertexAttribArray(
      TimelineRevisionsSharedResources.VBO_LAYOUT_LOCATION_INT_DYNAMIC_META,
    );
    gl.bindVertexArray(null);
    gl.bindBuffer(gl.ARRAY_BUFFER, null);
  }

  /**
   * Updates the dynamic buffer with highlight/selection status for each revision.
   * This is called when the user interacts with the timeline (hover, select).
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
    for (let i = 0; i < this.timeline.revisions.length; i++) {
      const selectionStatus =
        logElementHighlights[this.timeline.revisions[i].logIndex] ?? 0;
      const filterStatus = activeLogsIndices.has(
        this.timeline.revisions[i].logIndex,
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
   * Renders the revisions to the specified rectangle.
   *
   * @param gl The WebGL2 rendering context.
   * @param rect The screen rectangle where the revisions should be drawn.
   */
  renderColor(gl: WebGL2RenderingContext, rect: TimelineRect) {
    if (!this.revisionSharedResources.revisionLayerStylesUBOs) {
      return;
    }

    gl.enable(gl.DEPTH_TEST);
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA); // canvas expects premultiplied alpha, thus

    this.renderInstances(
      gl,
      this.revisionSharedResources.revisionsColorProgram,
      rect,
    );
  }

  /**
   * Renders the revisions for hit testing (off-screen).
   * Generates a color-coded map where each pixel corresponds to a revision index.
   *
   * @param gl The WebGL2 rendering context.
   * @param rect The screen rectangle where the hit test is being performed.
   */
  renderHittest(gl: WebGL2RenderingContext, rect: TimelineRect) {
    if (!this.revisionSharedResources.revisionLayerStylesUBOs) {
      return;
    }

    gl.disable(gl.BLEND);
    gl.enable(gl.DEPTH_TEST);

    this.renderInstances(
      gl,
      this.revisionSharedResources.revisionsHittestProgram,
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
    gl.bindVertexArray(this.revisionsVAO);

    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineRevisionsSharedResources.UBO_BINDING_VIEW_STATE,
      this.timelineSharedResources.uboViewState,
    );
    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineRevisionsSharedResources.UBO_BINDING_NUMBER_MSDF_ATLAS_PARAM,
      this.timelineSharedResources.uboNumberMSDFParamBuffer,
    );
    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_STYLES,
      this.revisionSharedResources.revisionStylesUBO,
    );
    gl.bindBufferBase(
      gl.UNIFORM_BUFFER,
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_LAYER_STYLES,
      this.revisionSharedResources.revisionLayerStylesUBOs[this.timeline.layer],
    );

    gl.activeTexture(gl.TEXTURE0);
    gl.bindTexture(
      gl.TEXTURE_2D,
      this.timelineSharedResources.numberMSDFTexture,
    );
    gl.bindSampler(0, this.timelineSharedResources.msdfSampler);
    gl.uniform1i(gl.getUniformLocation(program, 'numbersMSDFTexture'), 0);

    gl.activeTexture(gl.TEXTURE1);
    gl.bindTexture(
      gl.TEXTURE_2D,
      this.timelineSharedResources.iconsMSDFTexture,
    );
    gl.bindSampler(1, this.timelineSharedResources.msdfSampler);
    gl.uniform1i(gl.getUniformLocation(program, 'iconsMSDFTexture'), 1);

    gl.drawArraysInstanced(
      gl.TRIANGLE_STRIP,
      0,
      4,
      this.timeline.revisions.length,
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
    gl.deleteVertexArray(this.revisionsVAO);
    gl.deleteBuffer(this.timeVBO);
    gl.deleteBuffer(this.intStaticMetaVBO);
    gl.deleteBuffer(this.intDynamicMetaVBO);
  }
}

/**
 * Manages shared WebGL resources for rendering timeline revisions.
 * This includes shader programs, style buffers, and common constants.
 */
export class TimelineRevisionsSharedResources {
  public static readonly MAX_REVISION_STATE_TYPE = 128;

  public static readonly UBO_BINDING_VIEW_STATE = 0;
  public static readonly UBO_BINDING_NUMBER_MSDF_ATLAS_PARAM = 1;
  public static readonly UBO_BINDING_REVISION_STYLES = 2;
  public static readonly UBO_BINDING_REVISION_LAYER_STYLES = 3;

  public static readonly VBO_LAYOUT_LOCATION_TIME = 0;
  public static readonly VBO_LAYOUT_LOCATION_INT_STATIC_META = 1;
  public static readonly VBO_LAYOUT_LOCATION_INT_DYNAMIC_META = 2;

  public revisionsColorProgram!: WebGLProgram;

  public revisionsHittestProgram!: WebGLProgram;

  /**
   * An UBO storeing the styles for each revision state.
   */
  public revisionStylesUBO!: WebGLBuffer;

  /**
   * UBOs storeing styles to render revisions for each layer.
   * Styles bound to its revision states are stored in revisionStylesUBO.
   */
  public revisionLayerStylesUBOs!: { [key in TimelineLayer]: WebGLBuffer };

  private chartStyle!: TimelineChartStyle;

  private styleUpdated = false;

  constructor(
    private readonly timelineRendererSharedResources: TimelineRendererSharedResource,
  ) {}

  /**
   * Initializes shaders and buffers for revision rendering.
   *
   * @param gl The WebGL2 rendering context.
   */
  async setup(gl: WebGL2RenderingContext): Promise<void> {
    this.revisionsColorProgram = await WebGLUtil.compileAndLinkShaders(
      gl,
      'assets/revision-v2.vertex.glsl',
      'assets/revision-v2.color-fragment.glsl',
      {
        '#include "v2.shared.glsl"': 'assets/v2.shared.glsl',
        '#include "revision-v2.shared.glsl"': 'assets/revision-v2.shared.glsl',
      },
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsColorProgram,
      'ViewState',
      TimelineRevisionsSharedResources.UBO_BINDING_VIEW_STATE,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsColorProgram,
      'NumberMSDFAtlasParam',
      TimelineRevisionsSharedResources.UBO_BINDING_NUMBER_MSDF_ATLAS_PARAM,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsColorProgram,
      'RevisionStyles',
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_STYLES,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsColorProgram,
      'RevisionLayerStyles',
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_LAYER_STYLES,
    );

    this.revisionsHittestProgram = await WebGLUtil.compileAndLinkShaders(
      gl,
      'assets/revision-v2.vertex.glsl',
      'assets/revision-v2.hittest-fragment.glsl',
      {
        '#include "v2.shared.glsl"': 'assets/v2.shared.glsl',
        '#include "revision-v2.shared.glsl"': 'assets/revision-v2.shared.glsl',
      },
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsHittestProgram,
      'ViewState',
      TimelineRevisionsSharedResources.UBO_BINDING_VIEW_STATE,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsHittestProgram,
      'NumberMSDFAtlasParam',
      TimelineRevisionsSharedResources.UBO_BINDING_NUMBER_MSDF_ATLAS_PARAM,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsHittestProgram,
      'RevisionStyles',
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_STYLES,
    );
    WebGLUtil.setProgramUniformBlockBinding(
      gl,
      this.revisionsHittestProgram,
      'RevisionLayerStyles',
      TimelineRevisionsSharedResources.UBO_BINDING_REVISION_LAYER_STYLES,
    );
    this.revisionStylesUBO = gl.createBuffer();
  }

  /**
   * Updates the style buffers if the chart style has changed.
   * This includes updating colors, dimensions, and icon mappings for revision states.
   *
   * @param gl The WebGL2 rendering context.
   */
  beforeRender(gl: WebGL2RenderingContext) {
    if (this.styleUpdated) {
      this.revisionLayerStylesUBOs = {
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
      if (
        revisionStates.length >
        TimelineRevisionsSharedResources.MAX_REVISION_STATE_TYPE
      ) {
        throw new Error(
          'Too many revision states: Consider increassing the constant variables defined in the shader.',
        );
      }
      const uboSource = new Float32Array(
        TimelineRevisionsSharedResources.MAX_REVISION_STATE_TYPE * 12,
      );
      let baseOffset = 0;
      for (let i = 0; i < revisionStates.length; i++) {
        const color = revisionStatecolors[revisionStates[i]];
        uboSource[baseOffset + i * 4] = color[0];
        uboSource[baseOffset + i * 4 + 1] = color[1];
        uboSource[baseOffset + i * 4 + 2] = color[2];
        uboSource[baseOffset + i * 4 + 3] = 0;
      }
      baseOffset +=
        TimelineRevisionsSharedResources.MAX_REVISION_STATE_TYPE * 4;
      for (let i = 0; i < revisionStates.length; i++) {
        const rsm = RevisionStateMetadata[i];
        const iconCode = rsm.icon;
        if (iconCode !== '') {
          const iconUVSizes =
            this.timelineRendererSharedResources.getIconUVSizes(iconCode);
          uboSource[baseOffset + i * 4] = iconUVSizes[0];
          uboSource[baseOffset + i * 4 + 1] = iconUVSizes[1];
          uboSource[baseOffset + i * 4 + 2] = iconUVSizes[2];
          uboSource[baseOffset + i * 4 + 3] = iconUVSizes[3];
        }
      }
      baseOffset +=
        TimelineRevisionsSharedResources.MAX_REVISION_STATE_TYPE * 4;
      for (let i = 0; i < revisionStates.length; i++) {
        const rsm = RevisionStateMetadata[i];
        const revisionStateStyle =
          this.chartStyle.revisionStateStyle[rsm.style];
        uboSource[baseOffset + i * 4] = revisionStateStyle.alphaTransparency;
        uboSource[baseOffset + i * 4 + 1] =
          revisionStateStyle.borderStripePatten;
        uboSource[baseOffset + i * 4 + 2] =
          revisionStateStyle.bodyStripePattern;
        uboSource[baseOffset + i * 4 + 3] = 0;
      }

      gl.bindBuffer(gl.UNIFORM_BUFFER, this.revisionStylesUBO);
      gl.bufferData(gl.UNIFORM_BUFFER, uboSource, gl.STATIC_DRAW);
      gl.bindBuffer(gl.UNIFORM_BUFFER, null);
      this.styleUpdated = false;
    }
  }

  /**
   * Updates the chart style to be used for rendering.
   * This triggers a style update on the next frame.
   *
   * @param style The new timeline chart style.
   */
  updateChartStyle(style: TimelineChartStyle) {
    this.chartStyle = style;
    this.styleUpdated = true;
  }

  private createStyleUBOForLayer(
    gl: WebGL2RenderingContext,
    layer: TimelineLayer,
  ): WebGLBuffer {
    const revisionStyle = this.chartStyle.revisionStylesByLayer[layer];
    const selectionBorderColor = this.chartStyle.selectionBorderColor;
    const highlightBorderColor = this.chartStyle.highlightBorderColor;
    const ubo = gl.createBuffer();
    gl.bindBuffer(gl.UNIFORM_BUFFER, ubo);
    const timelineHeight = this.chartStyle.heightsByLayer[layer];
    const STD140_PADDING_FLOAT = 0;
    const bufferSource = new Float32Array([
      timelineHeight,
      revisionStyle.verticalPaddingInPx,
      revisionStyle.borderThickness,
      STD140_PADDING_FLOAT,
      revisionStyle.fontPaddingInPx[0], // 16
      revisionStyle.fontPaddingInPx[1],
      revisionStyle.fontSizeInPx,
      STD140_PADDING_FLOAT,
      revisionStyle.fontThicknessBySelectionType[ // 32
        TimelineChartItemHighlightType.None
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.fontThicknessBySelectionType[ // 48
        TimelineChartItemHighlightType.Hovered
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.fontThicknessBySelectionType[ // 64
        TimelineChartItemHighlightType.Selected
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.fontAntialias, // 80
      revisionStyle.fontStepInPx,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      selectionBorderColor[0], // 96
      selectionBorderColor[1],
      selectionBorderColor[2],
      revisionStyle.selectionBorderThickness,
      highlightBorderColor[0], // 112
      highlightBorderColor[1],
      highlightBorderColor[2],
      revisionStyle.hoverBorderThickness,
      revisionStyle.iconSizeInPx, // 128
      STD140_PADDING_FLOAT,
      revisionStyle.iconPaddingInPx[0], // 136
      revisionStyle.iconPaddingInPx[1],
      revisionStyle.iconThicknessBySelectionType[ // 144
        TimelineChartItemHighlightType.None
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.iconThicknessBySelectionType[ // 160
        TimelineChartItemHighlightType.Hovered
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.iconThicknessBySelectionType[ // 176
        TimelineChartItemHighlightType.Selected
      ],
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      STD140_PADDING_FLOAT,
      revisionStyle.iconAntialias, // 192
      this.chartStyle.borderStripePitch,
      this.chartStyle.bodyStripePitch,
      STD140_PADDING_FLOAT,
    ]);
    gl.bufferData(gl.UNIFORM_BUFFER, bufferSource, gl.STATIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);
    return ubo;
  }
}
