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
  GLContextManager,
  GLRenderer,
  WebGLContextLostException,
} from './glcontextmanager';

describe('GLContextManager', () => {
  let manager: GLContextManager<unknown>;
  let canvas: HTMLCanvasElement;
  let renderer: jasmine.SpyObj<GLRenderer<unknown>>;

  beforeEach(() => {
    canvas = document.createElement('canvas');
    renderer = jasmine.createSpyObj('GLRenderer', [
      'setup',
      'render',
      'resize',
    ]);
    manager = new GLContextManager(canvas, renderer);
  });

  afterEach(() => {
    manager.dispose();
  });

  describe('setup', () => {
    it('should initialize context and renderer', async () => {
      await manager.setup();

      expect(renderer.setup).toHaveBeenCalled();
      const glArgs = renderer.setup.calls.allArgs()[0][0];
      expect(glArgs).toBeInstanceOf(WebGL2RenderingContext);
    });
  });

  describe('render', () => {
    it('should call renderer.render if context is available', async () => {
      await manager.setup();
      manager.render({});
      expect(renderer.render).toHaveBeenCalled();
    });

    it('should not call renderer.render if context is not available', () => {
      manager.render({});
      expect(renderer.render).not.toHaveBeenCalled();
    });

    it('should handle WebGLContextLostException gracefully', async () => {
      await manager.setup();
      renderer.render.and.throwError(new WebGLContextLostException());
      spyOn(console, 'warn');
      manager.render({});
      expect(console.warn).toHaveBeenCalledWith(
        'WebGL2 context lost! Waiting context lost event to restore',
      );
    });
  });

  describe('context lost/restored (Real WebGL)', () => {
    it('should handle context lost and restore flow', async () => {
      await manager.setup();

      const gl = renderer.setup.calls.argsFor(0)[0];
      const ext = gl.getExtension('WEBGL_lose_context');

      if (!ext) {
        console.warn(
          'Skipping context lost test because WEBGL_lose_context is not supported',
        );
        return;
      }

      renderer.render.calls.reset();

      ext.loseContext(); // Simulate the context lost

      await new Promise((resolve) => setTimeout(resolve, 100)); // lose context is async. wait for the event to be propagated.

      // The render method shouldn't be called during context lost.
      manager.render({});
      expect(renderer.render).not.toHaveBeenCalled();

      renderer.setup.calls.reset();
      let setupResolver: () => void;
      const setupPromise = new Promise<void>((r) => (setupResolver = r));
      renderer.setup.and.callFake(() => setupPromise);

      ext.restoreContext();
      await new Promise((resolve) => setTimeout(resolve, 50)); // restore context is async. wait for the event to be propagated.

      // The setup function must be called after restore, but render shouldn't be called before the setup complete.
      expect(renderer.setup).toHaveBeenCalled();
      manager.render({});
      expect(renderer.render).not.toHaveBeenCalled();

      // Resolve the setup promise to allow the render to be called.
      setupResolver!();
      await new Promise((resolve) => setTimeout(resolve, 0));

      manager.render({});
      expect(renderer.render).toHaveBeenCalled();
    });
  });
});
