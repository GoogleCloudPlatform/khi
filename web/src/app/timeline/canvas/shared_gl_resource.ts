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

import {
  logTypeColors,
  logTypeDarkColors,
  logTypes,
  revisionStateDarkColors,
  revisionStatecolors,
  revisionStates,
  severities,
  severityBorderColors,
  severityColors,
} from 'src/app/generated';
import { GLResource } from './types';

/**
 * Initialize and hold gl resources commonly used from multiple timeline rows.
 */
export class SharedGLResources extends GLResource {
  /**
   * VAO for a rectangle(x,y in [-1 ,1]).
   * attribute 0: position, 3 dimentional FLOAT
   * index: UBYTE, 6 vertices
   */
  vaoRectangle!: WebGLVertexArrayObject;

  /**
   * A shader program for drawing instanced revision rectangles.
   */
  revisionShaderProgram!: WebGLProgram;

  /**
   * A shader program for drwaing instanced event rectangles.
   */
  eventShaderProgram!: WebGLProgram;

  /**
   * A shader program for drawing borders for each timeline rows.
   */
  borderShaderProgram!: WebGLProgram;

  /**
   * UBO containing the view status of timeline view status(scaling, offset of the times...etc).
   */
  uboViewState!: WebGLBuffer;

  /**
   * A texture holding numbers in tile.
   * The order is 0 to 9. height = 64px,width = 64px * 10.
   * texture index = 0
   */
  numberFontTexture!: WebGLTexture;

  /**
   * Color palette texture [(color type count)px X 1px] used in events
   * texture index = 1
   */
  logTypeColorPaletteTexture!: WebGLTexture;

  /**
   * Color palette texture [(color type count)px X 1px] used in revisions
   * texture index = 2
   */
  revisionStateColorPaletteTexture!: WebGLTexture;

  /**
   * Log severity color palette [(color type count)px X 1px] used in severities
   * texture index = 3
   */
  logSeverityColorPaletteTexture!: WebGLTexture;

  constructor(gl: WebGL2RenderingContext) {
    super(gl);
  }

  async init(): Promise<void> {
    this.ignoreGLContextLostExceptionAsync(async () => {
      this.uboViewState = this.glMust(this.gl.createBuffer());
      this.initShapes();
      await this.initShaders();
      this.initNumberTexture();
      this.initColorPaletteTextures();
    });
  }

  private initShapes() {
    const gl = this.gl;
    // Initialize rectangle
    // (layout=0 => position buffer)
    const vao = this.glMust(gl.createVertexArray());
    gl.bindVertexArray(vao);
    const vbo = this.glMust(gl.createBuffer());
    gl.bindBuffer(gl.ARRAY_BUFFER, vbo);
    // The order of position buffer
    // (0)---(1)
    //  |     |
    //  |     |
    // (3)---(2)
    //
    gl.bufferData(
      gl.ARRAY_BUFFER,
      new Float32Array([
        -1,
        1,
        0, //(0)
        1,
        1,
        0, //(1)
        1,
        -1,
        0, //(2)
        -1,
        -1,
        0, //(3)
      ]),
      gl.STATIC_DRAW,
    );
    gl.enableVertexAttribArray(0);
    gl.vertexAttribPointer(0, 3, gl.FLOAT, false, 0, 0);
    const ibo = this.glMust(gl.createBuffer());
    gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, ibo);
    gl.bufferData(
      gl.ELEMENT_ARRAY_BUFFER,
      new Uint8Array([0, 3, 1, 3, 2, 1]),
      gl.STATIC_DRAW,
    );
    gl.bindVertexArray(null);
    this.vaoRectangle = vao;
  }

  private async initShaders() {
    const gl = this.gl;

    // Init revision rectangle shader
    let program = await this.compileAndLinkShaders(
      'assets/revision.vertex.glsl',
      'assets/revision.fragment.glsl',
    );
    gl.uniformBlockBinding(
      program,
      gl.getUniformBlockIndex(program, 'ViewState'),
      0,
    );
    this.revisionShaderProgram = program;

    // Init event shader
    program = await this.compileAndLinkShaders(
      'assets/event.vertex.glsl',
      'assets/event.fragment.glsl',
    );
    gl.uniformBlockBinding(
      program,
      gl.getUniformBlockIndex(program, 'ViewState'),
      0,
    );
    this.eventShaderProgram = program;

    // Init border shader
    program = await this.compileAndLinkShaders(
      'assets/border.vertex.glsl',
      'assets/border.fragment.glsl',
    );
    gl.uniformBlockBinding(
      program,
      gl.getUniformBlockIndex(program, 'ViewState'),
      0,
    );
    this.borderShaderProgram = program;
  }

  private initNumberTexture() {
    const size = 256;
    const [numberCanvas, ctx2d] = this.generateCanvasForTextureSource(
      size * 10,
      size,
    );
    ctx2d.font = '240px serif';
    for (let digit = 0; digit < 10; digit++) {
      const measured = ctx2d.measureText('' + digit);
      ctx2d?.fillText(
        '' + digit,
        digit * size + (size - measured.width) / 2,
        measured.actualBoundingBoxAscent + size / 4,
        size,
      );
    }

    this.numberFontTexture = this.glMust(this.gl.createTexture());
    this.gl.activeTexture(this.gl.TEXTURE0);
    this.gl.bindTexture(this.gl.TEXTURE_2D, this.numberFontTexture);
    this.gl.texImage2D(
      this.gl.TEXTURE_2D,
      0,
      this.gl.RGBA,
      this.gl.RGBA,
      this.gl.UNSIGNED_BYTE,
      numberCanvas,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MIN_FILTER,
      this.gl.LINEAR_MIPMAP_LINEAR,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MAG_FILTER,
      this.gl.LINEAR,
    );
    this.gl.generateMipmap(this.gl.TEXTURE_2D);
  }

  private initColorPaletteTextures() {
    // event color palette
    const [eventColorPaletteCanvas, ctx2dForEvent] =
      this.generateCanvasForTextureSource(logTypes.length * 2, 1, false);
    eventColorPaletteCanvas.style.imageRendering = 'pixelated';

    for (let i = 0; i < logTypes.length; i++) {
      ctx2dForEvent.fillStyle = logTypeColors[logTypes[i]];
      ctx2dForEvent.fillRect(i * 2 + 0, 0, 1, 1);
      ctx2dForEvent.fillStyle = logTypeDarkColors[logTypes[i]];
      ctx2dForEvent.fillRect(i * 2 + 1, 0, 1, 1);
    }

    this.logTypeColorPaletteTexture = this.glMust(this.gl.createTexture());
    this.gl.activeTexture(this.gl.TEXTURE1);
    this.gl.bindTexture(this.gl.TEXTURE_2D, this.logTypeColorPaletteTexture);
    this.gl.texImage2D(
      this.gl.TEXTURE_2D,
      0,
      this.gl.RGBA,
      this.gl.RGBA,
      this.gl.UNSIGNED_BYTE,
      eventColorPaletteCanvas,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MIN_FILTER,
      this.gl.NEAREST,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MAG_FILTER,
      this.gl.NEAREST,
    );

    // revison state color palette
    const [revisionColorPaletteCanvas, ctx2dForRevision] =
      this.generateCanvasForTextureSource(revisionStates.length * 2, 1, false);
    revisionColorPaletteCanvas.style.imageRendering = 'pixelated';

    for (let i = 0; i < revisionStates.length; i++) {
      ctx2dForRevision.fillStyle = revisionStatecolors[revisionStates[i]];
      ctx2dForRevision.fillRect(i * 2 + 0, 0, 1, 1);
      ctx2dForRevision.fillStyle = revisionStateDarkColors[revisionStates[i]];
      ctx2dForRevision.fillRect(i * 2 + 1, 0, 1, 1);
    }

    this.revisionStateColorPaletteTexture = this.glMust(
      this.gl.createTexture(),
    );
    this.gl.activeTexture(this.gl.TEXTURE2);
    this.gl.bindTexture(
      this.gl.TEXTURE_2D,
      this.revisionStateColorPaletteTexture,
    );
    this.gl.texImage2D(
      this.gl.TEXTURE_2D,
      0,
      this.gl.RGBA,
      this.gl.RGBA,
      this.gl.UNSIGNED_BYTE,
      revisionColorPaletteCanvas,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MIN_FILTER,
      this.gl.NEAREST,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MAG_FILTER,
      this.gl.NEAREST,
    );

    // log severity color palette
    const [logSeverityColorPaletteCanvas, ctx2dForLogSeverity] =
      this.generateCanvasForTextureSource(severities.length, 2, false);
    logSeverityColorPaletteCanvas.style.imageRendering = 'pixelated';
    for (let i = 0; i < severities.length; i++) {
      ctx2dForLogSeverity.fillStyle = severityColors[severities[i]];
      ctx2dForLogSeverity.fillRect(i, 0, 1, 1);
      ctx2dForLogSeverity.fillStyle = severityBorderColors[severities[i]];
      ctx2dForLogSeverity.fillRect(i, 1, 1, 1);
    }
    this.logSeverityColorPaletteTexture = this.glMust(this.gl.createTexture());
    this.gl.activeTexture(this.gl.TEXTURE3);
    this.gl.bindTexture(
      this.gl.TEXTURE_2D,
      this.logSeverityColorPaletteTexture,
    );
    this.gl.texImage2D(
      this.gl.TEXTURE_2D,
      0,
      this.gl.RGBA,
      this.gl.RGBA,
      this.gl.UNSIGNED_BYTE,
      logSeverityColorPaletteCanvas,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MIN_FILTER,
      this.gl.NEAREST,
    );
    this.gl.texParameteri(
      this.gl.TEXTURE_2D,
      this.gl.TEXTURE_MAG_FILTER,
      this.gl.NEAREST,
    );
  }

  updateViewState(
    canvasWidth: number,
    canvasHeight: number,
    pixelPerTime: number,
    canvasPixelScale: number,
    relativeTimeOffset: number,
  ) {
    this.ignoreGLContextLostException(() => {
      const gl = this.gl;
      gl.bindBuffer(gl.UNIFORM_BUFFER, this.uboViewState);
      // viewState UBO structure
      // ---------------------------------------------------------------
      // |canvasWidth |canvasHeight|pixelPerTime|canvasPixelScale|timeOffset|logTypeCount
      // ---------------------------------------------------------------
      gl.bufferData(
        gl.UNIFORM_BUFFER,
        new Float32Array([
          canvasWidth,
          canvasHeight,
          pixelPerTime,
          canvasPixelScale,
          relativeTimeOffset,
          logTypes.length,
          0,
          0,
        ]),
        gl.STATIC_DRAW,
      );
      gl.bindBuffer(gl.UNIFORM_BUFFER, null);
    });
  }

  /**
   * Create a canvas element and obtain canvas 2d context.
   * With passing debug = true, the canvas will be appended at the end of body for debugging purpose.
   */
  private generateCanvasForTextureSource(
    width: number,
    height: number,
    debug = false,
  ): [HTMLCanvasElement, CanvasRenderingContext2D] {
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    if (debug) {
      canvas.style.scale = '10';
      document.body.appendChild(canvas);
    }
    return [canvas, canvas.getContext('2d')!];
  }

  public override dispose(): void {
    // TODO: Implement releasing used GL resources.
    // But this class is expected to be initialized at the first time, it won't be released.
    return;
  }
}
