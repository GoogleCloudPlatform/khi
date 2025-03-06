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

import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';

export const TIMELINE_ITEM_HEIGHTS: { [key: number]: number } = {
  [TimelineLayer.Kind]: 25,
  [TimelineLayer.Namespace]: 25,
  [TimelineLayer.Name]: 30,
  [TimelineLayer.Subresource]: 22,
};

/**
 * A base class using WebGL resources.
 */
export abstract class GLResource {
  private static readonly EXCEPTION_MESSAGE_GL_CONTEXT_LOST =
    'WebGL context lost';

  /**
   * If the GLContext lost at the last time or not. When this become true, the class needs to be recreated and GLResource is in the invalid status.
   */
  private hadLostGLContext = false;

  constructor(protected gl: WebGL2RenderingContext) {}

  /**
   * Check if the given result is null or not. Show gl error code when the result is null.
   */
  protected glMust<T>(result: T): NonNullable<T> {
    if (result === null || result === undefined) {
      if (
        this.gl.getError() === WebGL2RenderingContext.CONTEXT_LOST_WEBGL ||
        this.gl.isContextLost()
      ) {
        throw new Error(GLResource.EXCEPTION_MESSAGE_GL_CONTEXT_LOST);
      }
      throw new Error(
        'Failed to call webgl operation: code' + this.gl.getError(),
      );
    }
    return result;
  }

  protected async compileAndLinkShaders(
    vertexShaderPath: string,
    fragmentShaderPath: string,
  ): Promise<WebGLProgram> {
    const vertexShaderSource = await this.getShaderString(vertexShaderPath);
    const fragmentShaderSource = await this.getShaderString(fragmentShaderPath);
    // Vertex shader compilation
    const vs = this.glMust(this.gl.createShader(this.gl.VERTEX_SHADER));
    this.gl.shaderSource(vs, vertexShaderSource);
    this.gl.compileShader(vs);
    if (!this.gl.getShaderParameter(vs, this.gl.COMPILE_STATUS)) {
      console.error(`Compilation error on ${vertexShaderPath}`);
      console.error(this.gl.getShaderInfoLog(vs));
    }

    // Fragment shader compilation
    const fs = this.glMust(this.gl.createShader(this.gl.FRAGMENT_SHADER));
    this.gl.shaderSource(fs, fragmentShaderSource);
    this.gl.compileShader(fs);
    if (!this.gl.getShaderParameter(fs, this.gl.COMPILE_STATUS)) {
      console.error(`Compilation error on ${fragmentShaderPath}`);
      console.error(this.gl.getShaderInfoLog(fs));
    }

    // Links vertex/fragment shaders
    const program = this.glMust(this.gl.createProgram());
    this.gl.attachShader(program, vs);
    this.gl.attachShader(program, fs);
    this.gl.linkProgram(program);
    if (!this.gl.getProgramParameter(program, this.gl.LINK_STATUS)) {
      console.error(
        `Link error on ${vertexShaderPath} and ${fragmentShaderPath}`,
      );
      console.error(this.gl.getProgramInfoLog(program));
    }
    return program;
  }

  private async getShaderString(path: string): Promise<string> {
    const result = await fetch(path);
    return result.text();
  }

  /**
   * Ignores only the exception with the message GLResource.EXCEPTION_MESSAGE_GL_CONTEXT_LOST raised in the handler.
   * When the WebGL context lost, the context must be recreated outside of the class inheriting GLResource. GLResource class must ignore the error.
   */
  protected ignoreGLContextLostException<T>(handler: () => T): T | null {
    try {
      if (this.hadLostGLContext) return null;
      return handler();
    } catch (e) {
      if (
        e instanceof Error &&
        e.message === GLResource.EXCEPTION_MESSAGE_GL_CONTEXT_LOST
      ) {
        this.hadLostGLContext = true;
        return null;
      }
      throw e;
    }
  }

  /**
   * Async version of ignoreGLContextLostException.
   */
  protected async ignoreGLContextLostExceptionAsync<T>(
    handler: () => Promise<T>,
  ): Promise<T | null> {
    try {
      if (this.hadLostGLContext) return null;
      return await handler();
    } catch (e) {
      if (
        e instanceof Error &&
        e.message === GLResource.EXCEPTION_MESSAGE_GL_CONTEXT_LOST
      ) {
        this.hadLostGLContext = true;
        return null;
      }
      throw e;
    }
  }

  /**
   * Release allocated gl resources.
   */
  public abstract dispose(): void;
}

/**
 * CanvasSize represents a size of canvas element in px units.
 */
export interface CanvasSize {
  width: number;
  height: number;
}

/**
 * Represents the coordinate of mouse pointer.
 */
export interface CanvasMouseLocation {
  x: number;
  y: number;
}

export interface TimelineMouseLocation {
  timeline: ResourceTimeline;
  time: number;
  y: number;
}

/**
 * A rectangular region in the default coordinate space of OpenGL.
 */
export interface GLRect {
  left: number;
  bottom: number;
  width: number;
  height: number;
}
