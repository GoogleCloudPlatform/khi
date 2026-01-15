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

/**
 * Exception thrown when the WebGL context is lost.
 */
export class WebGLContextLostException extends Error {
  constructor(message: string = '') {
    super(`WebGL context lost: ${message}`);
  }
}

/**
 * Interface for a component that renders content using WebGL.
 * Implementers are responsible for handling the low-level WebGL commands.
 */
export interface GLRenderer<RenderArgs> {
  /**
   * Called when the canvas is resized.
   * @param width The new width of the canvas in CSS pixels.
   * @param height The new height of the canvas in CSS pixels.
   * @param devicePixelRatio The current device pixel ratio.
   */
  resize(width: number, height: number, devicePixelRatio: number): void;

  /**
   * Called to initialize WebGL resources (shaders, buffers, etc.).
   * This may be called multiple times if the WebGL context is lost and restored.
   * @param gl The WebGL rendering context.
   */
  setup(gl: WebGLRenderingContext): Promise<void>;

  /**
   * Called to render a frame.
   * @param gl The WebGL rendering context.
   * @param args The arguments passed to the render function, containing frame-specific data.
   */
  render(gl: WebGLRenderingContext, args: RenderArgs): void;
}
/**
 * GLContextManager provides customized context handling for WebGL2.
 *
 * WebGL context may be lost in the middle of rendering and renderers may throw WebGLContextLostException.
 * This class catches the context lost exception, handles context lost and restored events,
 * and automatically restores the context and resources.
 */
export class GLContextManager<RenderArgs> {
  private gl: WebGL2RenderingContext | null = null;

  private abortController = new AbortController();

  /**
   * Creates a new GLContextManager.
   * @param canvas The HTMLCanvasElement to manage the context for.
   * @param renderer The renderer implementation to define what to draw.
   */
  constructor(
    private canvas: HTMLCanvasElement,
    private renderer: GLRenderer<RenderArgs>,
  ) {
    let restoringContext: WebGL2RenderingContext | null = null;
    this.canvas.addEventListener(
      'webglcontextlost',
      (e) => {
        e.preventDefault();
        restoringContext = this.gl;
        console.warn('WebGL2 context lost! Attempting to restore...');
        this.gl = null;
      },
      { signal: this.abortController.signal },
    );
    this.canvas.addEventListener(
      'webglcontextrestored',
      async () => {
        if (!restoringContext)
          throw new Error(
            'unreachable. Context restored event is called without a context lost event.',
          );
        console.warn('WebGL2 context restored. Recreating gl resources again.');
        await this.trySetup(restoringContext);
        this.gl = restoringContext;
        console.info('GL resource recreation completed.');
      },
      { signal: this.abortController.signal },
    );
  }

  /**
   * Initializes the WebGL context and sets up the renderer.
   * Should be called once after construction.
   */
  async setup() {
    const gl = this.canvas.getContext('webgl2');
    if (!gl) {
      alert(
        'Failed to obtain the WebGL2 context. Please try reload this page or restart your computer.',
      );
      return;
    }
    gl.drawingBufferColorSpace = 'display-p3';
    await this.trySetup(gl);
    this.gl = gl;
    console.info('GL context setup completed.');
  }

  /**
   * Renders a frame using the underlying renderer.
   * @param args Arguments to pass to the renderer.
   */
  render(args: RenderArgs) {
    if (this.gl === null) return;
    this.tryRender(this.gl, args);
  }

  /**
   * Disposes of the context manager and removes event listeners.
   */
  dispose() {
    this.abortController.abort();
  }

  private async trySetup(gl: WebGL2RenderingContext) {
    try {
      await this.renderer.setup(gl);
    } catch (e) {
      if (e instanceof WebGLContextLostException) {
        console.warn(
          'WebGL2 context lost! Waiting context lost event to restore',
        );
      } else {
        throw e;
      }
    }
  }

  private tryRender(gl: WebGL2RenderingContext, args: RenderArgs) {
    try {
      this.renderer.render(gl, args);
    } catch (e) {
      if (e instanceof WebGLContextLostException) {
        console.warn(
          'WebGL2 context lost! Waiting context lost event to restore',
        );
      } else {
        throw e;
      }
    }
  }
}
