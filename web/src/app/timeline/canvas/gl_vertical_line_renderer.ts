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

import { SharedGLResources } from './shared_gl_resource';
import { CanvasSize, GLResource } from './types';

/**
 * A renderer to draw a single vertical line.
 * Expected to be used for drawing the selection line.
 */
export class GLVerticalLineRenderer extends GLResource {
  private verticalLineProgram!: WebGLProgram;

  private lineStateUBO!: WebGLBuffer;

  private lineStateUBOSource: Float32Array = new Float32Array(9);

  constructor(gl: WebGL2RenderingContext) {
    super(gl);
  }

  public async init() {
    this.ignoreGLContextLostExceptionAsync(async () => {
      this.initBuffers();
      await this.initProgram();
    });
  }

  private initBuffers() {
    this.lineStateUBO = this.glMust(this.gl.createBuffer());
  }

  private async initProgram(): Promise<void> {
    this.verticalLineProgram = await this.compileAndLinkShaders(
      'assets/vertical-line.vertex.glsl',
      'assets/vertical-line.fragment.glsl',
    );
    this.gl.uniformBlockBinding(
      this.verticalLineProgram,
      this.gl.getUniformBlockIndex(this.verticalLineProgram, 'ViewState'),
      0,
    );
    this.gl.uniformBlockBinding(
      this.verticalLineProgram,
      this.gl.getUniformBlockIndex(this.verticalLineProgram, 'LineState'),
      1,
    );
  }

  public render(
    canvasSize: CanvasSize,
    canvasPixelRatio: number,
    sharedResource: SharedGLResources,
    tsOffset: number,
    thicknessInPixels: number,
    color: number[],
  ) {
    this.ignoreGLContextLostException(() => {
      this.gl.blendFunc(this.gl.SRC_ALPHA, this.gl.ONE_MINUS_SRC_ALPHA);
      this.gl.viewport(
        0,
        0,
        canvasSize.width * canvasPixelRatio,
        canvasSize.height * canvasPixelRatio,
      );

      // Update LineState uniform buffer
      this.lineStateUBOSource[0] = tsOffset;
      this.lineStateUBOSource[1] = thicknessInPixels;
      this.lineStateUBOSource[4] = color[0];
      this.lineStateUBOSource[5] = color[1];
      this.lineStateUBOSource[6] = color[2];
      this.lineStateUBOSource[7] = color[3];

      this.gl.bindBuffer(this.gl.UNIFORM_BUFFER, this.lineStateUBO);
      this.gl.bufferData(
        this.gl.UNIFORM_BUFFER,
        this.lineStateUBOSource,
        this.gl.STATIC_DRAW,
      );
      this.gl.bindBuffer(this.gl.UNIFORM_BUFFER, null);

      // Draw a line
      this.gl.disable(this.gl.SCISSOR_TEST);
      this.gl.enable(this.gl.DEPTH_TEST);
      this.gl.depthMask(false);
      this.gl.useProgram(this.verticalLineProgram);
      this.gl.bindVertexArray(sharedResource.vaoRectangle);
      this.gl.bindBufferBase(
        this.gl.UNIFORM_BUFFER,
        0,
        sharedResource.uboViewState,
      );
      this.gl.bindBufferBase(this.gl.UNIFORM_BUFFER, 1, this.lineStateUBO);

      this.gl.drawElements(this.gl.TRIANGLES, 6, this.gl.UNSIGNED_BYTE, 0);
      this.gl.flush();
    });
  }

  public override dispose(): void {
    this.gl.deleteProgram(this.verticalLineProgram);
  }
}
