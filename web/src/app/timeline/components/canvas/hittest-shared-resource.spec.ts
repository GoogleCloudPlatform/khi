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
  TimelineHitTestSharedResource,
  ScissorRect,
} from 'src/app/timeline/components/canvas/hittest-shared-resource';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';

describe('TimelineHitTestSharedResource', () => {
  let hittestResource: TimelineHitTestSharedResource;
  let canvas: HTMLCanvasElement;
  let gl: WebGL2RenderingContext;

  beforeEach(() => {
    canvas = document.createElement('canvas');
    canvas.width = 100;
    canvas.height = 100;
    const context = canvas.getContext('webgl2');
    if (!context) {
      throw new Error('WebGL2 context is not supported in this environment');
    }
    gl = context;
    hittestResource = new TimelineHitTestSharedResource();
  });

  describe('setup', () => {
    it('should initialize WebGL framebuffer and texture', () => {
      hittestResource.setup(gl);

      expect(hittestResource.hittestFBO).toBeTruthy();
      expect(hittestResource.hittestTexture).toBeTruthy();
    });
  });

  describe('beforeRender and afterRender', () => {
    it('should configure scissor test, clear buffer, and restore state', () => {
      hittestResource.setup(gl);
      hittestResource.resize(100, 100);

      const scissorRect: ScissorRect = {
        x: 10,
        y: 20,
        width: 1,
        height: 1,
      };

      spyOn(gl, 'enable').and.callThrough();
      spyOn(gl, 'scissor').and.callThrough();
      spyOn(gl, 'clearBufferuiv').and.callThrough();
      spyOn(gl, 'disable').and.callThrough();

      hittestResource.beforeRender(gl, scissorRect);

      expect(gl.enable).toHaveBeenCalledWith(gl.SCISSOR_TEST);
      expect(gl.scissor).toHaveBeenCalledWith(10, 20, 1, 1);
      expect(gl.clearBufferuiv).toHaveBeenCalled();

      hittestResource.afterRender(gl);

      expect(gl.disable).toHaveBeenCalledWith(gl.SCISSOR_TEST);
    });
  });

  describe('hittest', () => {
    it('should return hit test result for a given coordinate', async () => {
      hittestResource.setup(gl);
      hittestResource.resize(100, 100);

      const mockData = await createMockInspectionData();
      const mockTimeline = mockData.timelineStore.timelines[0];

      const result = hittestResource.hittest(gl, 10, 20, mockTimeline);

      expect(result.clientX).toBe(10);
      expect(result.clientY).toBe(20);
      expect(result.timeline).toBe(mockTimeline);
    });
  });
});
