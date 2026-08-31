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

import { BMFontConfig } from 'src/app/store/domain/style';
import {
  SharedTmpBuffer,
  WebGLUtil,
} from 'src/app/timeline/components/canvas/glutil';

describe('glutil', () => {
  describe('WebGLUtil', () => {
    let originalFetch: typeof window.fetch;

    beforeEach(() => {
      originalFetch = window.fetch;
      WebGLUtil.clearCache();
    });

    afterEach(() => {
      window.fetch = originalFetch;
      WebGLUtil.clearCache();
    });

    describe('getShaderString', () => {
      it('fetches shader source using the provided path directly without prepending slash', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.returnValue(
          Promise.resolve(
            new Response('void main() {}', {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        const shaderSource =
          await WebGLUtil.getShaderString('assets/test.glsl');

        expect(fetchSpy).toHaveBeenCalledWith('assets/test.glsl');
        expect(shaderSource).toBe('void main() {}');
      });

      it('resolves relative shader paths under the document baseURI when a custom base href is present', async () => {
        const baseElement = document.createElement('base');
        baseElement.href = 'http://localhost/subpath/khi/';
        document.head.appendChild(baseElement);

        try {
          let requestedUrl = '';
          const fetchSpy = jasmine
            .createSpy('fetch')
            .and.callFake((input: RequestInfo | URL) => {
              const req = new Request(input);
              requestedUrl = req.url;
              return Promise.resolve(
                new Response('void main() {}', {
                  status: 200,
                  statusText: 'OK',
                }),
              );
            });
          window.fetch = fetchSpy;

          const result = await WebGLUtil.getShaderString('assets/test.glsl');

          expect(fetchSpy).toHaveBeenCalledWith('assets/test.glsl');
          expect(requestedUrl).toBe(
            'http://localhost/subpath/khi/assets/test.glsl',
          );
          expect(result).toBe('void main() {}');
        } finally {
          document.head.removeChild(baseElement);
        }
      });

      it('caches the result for identical paths and does not issue duplicate fetch calls', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.returnValue(
          Promise.resolve(
            new Response('precision highp float;', {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        const firstResult =
          await WebGLUtil.getShaderString('assets/shared.glsl');
        const secondResult =
          await WebGLUtil.getShaderString('assets/shared.glsl');

        expect(fetchSpy).toHaveBeenCalledTimes(1);
        expect(firstResult).toBe('precision highp float;');
        expect(secondResult).toBe('precision highp float;');
      });

      it('reuses in-flight promise for concurrent requests to the same path', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.returnValue(
          Promise.resolve(
            new Response('uniform ViewState { mat4 u_vp; };', {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        const [firstResult, secondResult] = await Promise.all([
          WebGLUtil.getShaderString('assets/concurrent.glsl'),
          WebGLUtil.getShaderString('assets/concurrent.glsl'),
        ]);

        expect(fetchSpy).toHaveBeenCalledTimes(1);
        expect(firstResult).toBe('uniform ViewState { mat4 u_vp; };');
        expect(secondResult).toBe('uniform ViewState { mat4 u_vp; };');
      });

      it('throws an error and evicts the path from cache on fetch failure', async () => {
        let callCount = 0;
        const fetchSpy = jasmine.createSpy('fetch').and.callFake(async () => {
          callCount++;
          if (callCount === 1) {
            return new Response(null, {
              status: 404,
              statusText: 'Not Found',
            });
          }
          return new Response('recovered shader', {
            status: 200,
            statusText: 'OK',
          });
        });
        window.fetch = fetchSpy;

        await expectAsync(
          WebGLUtil.getShaderString('assets/not-found.glsl'),
        ).toBeRejectedWithError(
          /Failed to load shader file at assets\/not-found.glsl: HTTP 404 Not Found/,
        );

        const retryResult = await WebGLUtil.getShaderString(
          'assets/not-found.glsl',
        );
        expect(fetchSpy).toHaveBeenCalledTimes(2);
        expect(retryResult).toBe('recovered shader');
      });

      it('clears cached shaders when clearCache is called', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.callFake(() =>
          Promise.resolve(
            new Response('shader content', {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        await WebGLUtil.getShaderString('assets/test.glsl');
        WebGLUtil.clearCache();
        await WebGLUtil.getShaderString('assets/test.glsl');

        expect(fetchSpy).toHaveBeenCalledTimes(2);
      });
    });

    describe('loadBMFontConfig', () => {
      const mockConfig: BMFontConfig = {
        pages: ['assets/font.png'],
        common: {
          lineHeight: 16,
          base: 12,
          scaleW: 128,
          scaleH: 128,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
        chars: [],
      };

      it('fetches and parses BMFontConfig JSON', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.returnValue(
          Promise.resolve(
            new Response(JSON.stringify(mockConfig), {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        const config = await WebGLUtil.loadBMFontConfig('assets/font.json');
        expect(fetchSpy).toHaveBeenCalledWith('assets/font.json');
        expect(config).toEqual(mockConfig);
      });

      it('resolves relative BMFont paths under the document baseURI when a custom base href is present', async () => {
        const baseElement = document.createElement('base');
        baseElement.href = 'http://localhost/subpath/khi/';
        document.head.appendChild(baseElement);

        try {
          let requestedUrl = '';
          const fetchSpy = jasmine
            .createSpy('fetch')
            .and.callFake((input: RequestInfo | URL) => {
              const req = new Request(input);
              requestedUrl = req.url;
              return Promise.resolve(
                new Response(JSON.stringify(mockConfig), {
                  status: 200,
                  statusText: 'OK',
                }),
              );
            });
          window.fetch = fetchSpy;

          const config = await WebGLUtil.loadBMFontConfig('assets/font.json');

          expect(fetchSpy).toHaveBeenCalledWith('assets/font.json');
          expect(requestedUrl).toBe(
            'http://localhost/subpath/khi/assets/font.json',
          );
          expect(config).toEqual(mockConfig);
        } finally {
          document.head.removeChild(baseElement);
        }
      });

      it('caches the result for identical BMFont config paths', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.returnValue(
          Promise.resolve(
            new Response(JSON.stringify(mockConfig), {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        const firstResult =
          await WebGLUtil.loadBMFontConfig('assets/font.json');
        const secondResult =
          await WebGLUtil.loadBMFontConfig('assets/font.json');

        expect(fetchSpy).toHaveBeenCalledTimes(1);
        expect(firstResult).toEqual(mockConfig);
        expect(secondResult).toEqual(mockConfig);
      });

      it('throws an error and evicts the BMFont path from cache on failure', async () => {
        let callCount = 0;
        const fetchSpy = jasmine.createSpy('fetch').and.callFake(async () => {
          callCount++;
          if (callCount === 1) {
            return new Response(null, {
              status: 500,
              statusText: 'Internal Server Error',
            });
          }
          return new Response(JSON.stringify(mockConfig), {
            status: 200,
            statusText: 'OK',
          });
        });
        window.fetch = fetchSpy;

        await expectAsync(
          WebGLUtil.loadBMFontConfig('assets/font-error.json'),
        ).toBeRejectedWithError(
          /Failed to load BMFont config at assets\/font-error.json: HTTP 500 Internal Server Error/,
        );

        const retryResult = await WebGLUtil.loadBMFontConfig(
          'assets/font-error.json',
        );
        expect(fetchSpy).toHaveBeenCalledTimes(2);
        expect(retryResult).toEqual(mockConfig);
      });
    });

    describe('loadTexture and clearCache', () => {
      let canvas: HTMLCanvasElement;
      let gl: WebGL2RenderingContext;

      beforeEach(() => {
        canvas = document.createElement('canvas');
        const context = canvas.getContext('webgl2');
        if (!context) {
          fail('WebGL2 context is not supported in this environment');
          return;
        }
        gl = context;
      });

      it('caches loaded image when loadTexture is called for the same path', async () => {
        const sampleImageUri =
          'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==';

        const texture1 = await WebGLUtil.loadTexture(gl, sampleImageUri);
        const texture2 = await WebGLUtil.loadTexture(gl, sampleImageUri);

        expect(texture1).toBeTruthy();
        expect(texture2).toBeTruthy();

        gl.deleteTexture(texture1);
        gl.deleteTexture(texture2);
      });

      it('clears all caches when clearCache is called', async () => {
        const fetchSpy = jasmine.createSpy('fetch').and.callFake(() =>
          Promise.resolve(
            new Response('sample', {
              status: 200,
              statusText: 'OK',
            }),
          ),
        );
        window.fetch = fetchSpy;

        await WebGLUtil.getShaderString('assets/clear-test.glsl');
        WebGLUtil.clearCache();
        await WebGLUtil.getShaderString('assets/clear-test.glsl');

        expect(fetchSpy).toHaveBeenCalledTimes(2);
      });
    });

    describe('setProgramUniformBlockBinding', () => {
      let canvas: HTMLCanvasElement;
      let gl: WebGL2RenderingContext;
      let program: WebGLProgram;

      beforeEach(() => {
        canvas = document.createElement('canvas');
        const context = canvas.getContext('webgl2');
        if (!context) {
          fail('WebGL2 context is not supported in this environment');
          return;
        }
        gl = context;
        program = gl.createProgram()!;
      });

      afterEach(() => {
        if (gl && program) {
          gl.deleteProgram(program);
        }
      });

      it('throws an error if uniform block is not found', () => {
        spyOn(gl, 'getUniformBlockIndex').and.returnValue(-1);

        expect(() => {
          WebGLUtil.setProgramUniformBlockBinding(
            gl,
            program,
            'NonExistentBlock',
            1,
          );
        }).toThrowError('Uniform block NonExistentBlock not found');
      });

      it('binds uniform block index if block exists', () => {
        spyOn(gl, 'getUniformBlockIndex').and.returnValue(3);
        const bindSpy = spyOn(gl, 'uniformBlockBinding');

        WebGLUtil.setProgramUniformBlockBinding(gl, program, 'ValidBlock', 2);

        expect(bindSpy).toHaveBeenCalledWith(program, 3, 2);
      });
    });
  });

  describe('SharedTmpBuffer', () => {
    it('allocates and returns typed array views', () => {
      const buffer = new SharedTmpBuffer(1024);
      const f32 = buffer.float32Array(10);
      expect(f32.length).toBe(10);
      expect(f32.byteLength).toBe(40);

      const u32 = buffer.uint32Array(5);
      expect(u32.length).toBe(5);
      expect(u32.byteLength).toBe(20);

      const dv = buffer.dataView(64);
      expect(dv.byteLength).toBe(64);
    });

    it('grows buffer when requested size exceeds current capacity', () => {
      const buffer = new SharedTmpBuffer(16);
      const largeArray = buffer.float32Array(100);
      expect(largeArray.length).toBe(100);
      expect(largeArray.buffer.byteLength).toBeGreaterThanOrEqual(400);
    });

    it('does not allocate a smaller buffer when smaller size is requested afterwards', () => {
      const buffer = new SharedTmpBuffer(16);
      const largeArray = buffer.float32Array(100);
      const allocatedCapacity = largeArray.buffer.byteLength;

      const smallArray = buffer.float32Array(10);
      expect(smallArray.length).toBe(10);
      expect(smallArray.buffer.byteLength).toBe(allocatedCapacity);
    });
  });
});
