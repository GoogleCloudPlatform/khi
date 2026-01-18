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

import { ResourceTimeline } from 'src/app/store/timeline';
import { WebGLContextLostException } from './glcontextmanager';

/**
 * Result of a hit test operation on the timeline.
 */
export interface HitTestResult {
  /** The timeline that was hit, or null if no timeline was hit. */
  timeline: ResourceTimeline | null;
  /** The index of the revision that was hit, if applicable. */
  revisionIndex?: number;
  /** The index of the event that was hit, if applicable. */
  eventIndex?: number;
  /** The X coordinate of the mouse position. */
  clientX: number;
  /** The Y coordinate of the mouse position. */
  clientY: number;
}

/**
 * Shared resource for handling hit testing on timelines using WebGL.
 * It manages a framebuffer object (FBO) and texture to render an index map
 * for efficient retrieval of the object at a specific pixel.
 */
export class TimelineHitTestSharedResource {
  public hittestFBO!: WebGLFramebuffer;

  /**
   * hittestTexture is the color buffer texture bound to hittestFBO.
   * RG32UI : r: index of the item, g: type of the item(1: revision, 2: event)
   */
  public hittestTexture!: WebGLTexture;

  private depthRenderBuffer!: WebGLRenderbuffer;

  private width = 1;
  private height = 1;
  private sizeUpdated = false;

  /**
   * Initializes the framebuffer and texture resources for hit testing.
   * @param gl The WebGL2 rendering context.
   */
  setup(gl: WebGL2RenderingContext) {
    this.hittestFBO = gl.createFramebuffer();
    if (this.hittestFBO === null) {
      throw new WebGLContextLostException('Failed to create framebuffer');
    }
    this.hittestTexture = gl.createTexture();
    if (this.hittestTexture === null) {
      throw new WebGLContextLostException('Failed to create texture');
    }
    gl.bindTexture(gl.TEXTURE_2D, this.hittestTexture);
    gl.texImage2D(
      gl.TEXTURE_2D,
      0,
      gl.RG32UI,
      this.width,
      this.height,
      0,
      gl.RG_INTEGER,
      gl.UNSIGNED_INT,
      null,
    );
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
    gl.bindTexture(gl.TEXTURE_2D, null);
    this.depthRenderBuffer = gl.createRenderbuffer();
    gl.bindRenderbuffer(gl.RENDERBUFFER, this.depthRenderBuffer);
    gl.renderbufferStorage(
      gl.RENDERBUFFER,
      gl.DEPTH_COMPONENT24,
      this.width,
      this.height,
    );
    gl.bindRenderbuffer(gl.RENDERBUFFER, null);
    gl.bindFramebuffer(gl.FRAMEBUFFER, this.hittestFBO);
    gl.framebufferTexture2D(
      gl.FRAMEBUFFER,
      gl.COLOR_ATTACHMENT0,
      gl.TEXTURE_2D,
      this.hittestTexture,
      0,
    );
    gl.framebufferRenderbuffer(
      gl.FRAMEBUFFER,
      gl.DEPTH_ATTACHMENT,
      gl.RENDERBUFFER,
      this.depthRenderBuffer,
    );
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
  }

  /**
   * Prepares the shared resource for rendering the hit test map.
   * Binds the FBO and clears the buffers. Resizes the texture if necessary.
   * @param gl The WebGL2 rendering context.
   */
  beforeRender(gl: WebGL2RenderingContext) {
    if (this.sizeUpdated) {
      gl.bindTexture(gl.TEXTURE_2D, this.hittestTexture);
      gl.texImage2D(
        gl.TEXTURE_2D,
        0,
        gl.RG32UI,
        this.width,
        this.height,
        0,
        gl.RG_INTEGER,
        gl.UNSIGNED_INT,
        null,
      );
      gl.bindTexture(gl.TEXTURE_2D, null);
      gl.bindRenderbuffer(gl.RENDERBUFFER, this.depthRenderBuffer);
      gl.renderbufferStorage(
        gl.RENDERBUFFER,
        gl.DEPTH_COMPONENT24,
        this.width,
        this.height,
      );
      gl.bindRenderbuffer(gl.RENDERBUFFER, null);
      this.sizeUpdated = false;
    }
    gl.bindFramebuffer(gl.FRAMEBUFFER, this.hittestFBO);
    gl.clearBufferuiv(gl.COLOR, 0, new Uint32Array([0, 0, 0, 0]));
    gl.clear(gl.DEPTH_BUFFER_BIT);
  }

  /**
   * Unbinds the framebuffer after rendering is complete.
   * @param gl The WebGL2 rendering context.
   */
  afterRender(gl: WebGL2RenderingContext) {
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
  }

  /**
   * Updates the dimensions of the hit test texture and buffer.
   * @param width The new width in pixels.
   * @param height The new height in pixels.
   */
  resize(width: number, height: number) {
    if (width <= 0 || height <= 0) {
      return;
    }
    this.width = width;
    this.height = height;
    this.sizeUpdated = true;
  }

  /**
   * Performs a hit test at the specified coordinates by reading from the framebuffer.
   *
   * @param gl The WebGL2 rendering context.
   * @param x The X coordinate of the hit test.
   * @param y The Y coordinate of the hit test.
   * @param timeline The timeline context for the hit test result.
   * @returns A HitTestResult object containing details about the hit.
   */
  hittest(
    gl: WebGL2RenderingContext,
    x: number,
    y: number,
    timeline: ResourceTimeline,
  ): HitTestResult {
    const pixels = new Uint32Array(2);
    gl.bindFramebuffer(gl.FRAMEBUFFER, this.hittestFBO);
    gl.readPixels(
      x,
      this.height - y - 1,
      1,
      1,
      gl.RG_INTEGER,
      gl.UNSIGNED_INT,
      pixels,
    );
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
    const index = pixels[0];
    const type = pixels[1];
    const base: HitTestResult = {
      timeline,
      clientX: x,
      clientY: y,
    };
    switch (type) {
      case 1:
        return {
          ...base,
          revisionIndex: index,
        };
      case 2:
        return {
          ...base,
          eventIndex: index,
        };
      default:
        return {
          ...base,
        };
    }
  }
}
