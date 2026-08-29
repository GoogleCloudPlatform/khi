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
  GLSLDataType,
  GLSLFunctionTester,
} from 'src/app/timeline/components/canvas/glsl-function-tester';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';

describe('GLSLFunctionTester', () => {
  let tester: GLSLFunctionTester;

  beforeEach(() => {
    tester = new GLSLFunctionTester();
    tester.setup();
  });

  afterEach(() => {
    tester.dispose();
  });

  describe('basic expression evaluation', () => {
    it('evaluates float arithmetic expressions', async () => {
      const result = await tester.runExpression<number>({
        expression: '12.5 + 7.25',
        returnType: GLSLDataType.Float,
      });
      expect(result).toBeCloseTo(19.75, 4);
    });

    it('evaluates int and uint expressions', async () => {
      const intResult = await tester.runExpression<number>({
        expression: '10 - 25',
        returnType: GLSLDataType.Int,
      });
      expect(intResult).toBe(-15);

      const uintResult = await tester.runExpression<number>({
        expression: '100u * 30u',
        returnType: GLSLDataType.Uint,
      });
      expect(uintResult).toBe(3000);
    });

    it('evaluates boolean logic expressions', async () => {
      const trueResult = await tester.runExpression<boolean>({
        expression: '(5u > 3u) && (10u != 0u)',
        returnType: GLSLDataType.Bool,
      });
      expect(trueResult).toBe(true);

      const falseResult = await tester.runExpression<boolean>({
        expression: '42.0 < 10.0',
        returnType: GLSLDataType.Bool,
      });
      expect(falseResult).toBe(false);
    });

    it('evaluates vec2, vec3, and vec4 expressions', async () => {
      const vec2Result = await tester.runExpression<number[]>({
        expression: 'vec2(1.5, 2.5) * 2.0',
        returnType: GLSLDataType.Vec2,
      });
      expect(vec2Result[0]).toBeCloseTo(3.0, 4);
      expect(vec2Result[1]).toBeCloseTo(5.0, 4);

      const vec4Result = await tester.runExpression<number[]>({
        expression: 'vec4(1.0, 0.5, 0.25, 1.0)',
        returnType: GLSLDataType.Vec4,
      });
      expect(vec4Result).toEqual([1.0, 0.5, 0.25, 1.0]);
    });

    it('evaluates uvec4 expressions', async () => {
      const uvec4Result = await tester.runExpression<number[]>({
        expression: 'uvec4(1u, 2u, 100u, 200u)',
        returnType: GLSLDataType.UVec4,
      });
      expect(uvec4Result).toEqual([1, 2, 100, 200]);
    });

    it('passes uniforms correctly to expressions', async () => {
      const result = await tester.runExpression<number>({
        declarations: 'uniform float u_multiplier;',
        expression: '10.0 * u_multiplier',
        returnType: GLSLDataType.Float,
        uniforms: [
          {
            name: 'u_multiplier',
            setter: (gl, loc) => gl.uniform1f(loc, 3.5),
          },
        ],
      });
      expect(result).toBeCloseTo(35.0, 4);
    });
  });

  describe('shared.glsl functions and constants', () => {
    it('provides correct mathematical constants PI and SQRT2', async () => {
      const piResult = await tester.runExpression<number>({
        expression: 'float(PI)',
        returnType: GLSLDataType.Float,
        includes: {
          '#include "shared.glsl"': 'assets/shared.glsl',
        },
      });
      expect(piResult).toBeCloseTo(Math.PI, 4);

      const sqrt2Result = await tester.runExpression<number>({
        expression: 'float(SQRT2)',
        returnType: GLSLDataType.Float,
        includes: {
          '#include "shared.glsl"': 'assets/shared.glsl',
        },
      });
      expect(sqrt2Result).toBeCloseTo(Math.SQRT2, 4);
    });

    it('defines BITSET_TEXTURE_WIDTH as 2048u', async () => {
      const bitsetWidth = await tester.runExpression<number>({
        expression: 'BITSET_TEXTURE_WIDTH',
        returnType: GLSLDataType.Uint,
        includes: {
          '#include "shared.glsl"': 'assets/shared.glsl',
        },
      });
      expect(bitsetWidth).toBe(2048);
    });

    describe('checkBitset function', () => {
      it('correctly checks bits across word boundaries and multiple texture rows', async () => {
        const bitset = IdBitset.createEmpty();
        // Word 0
        bitset.add(0);
        bitset.add(1);
        bitset.add(31);
        // Word 1
        bitset.add(32);
        bitset.add(42);
        // Word 2047 (last word of row 0: 2047 * 32 = 65504 to 65535)
        bitset.add(65535);
        // Word 2048 (first word of row 1: 2048 * 32 = 65536)
        bitset.add(65536);
        bitset.add(70000);
        bitset.add(100000);

        const bitsetTexture = tester.createBitsetTexture(bitset.words, 2048);

        const checkBit = async (id: number): Promise<boolean> => {
          return tester.runExpression<boolean>({
            declarations: 'uniform uint u_targetId;',
            expression: 'checkBitset(u_testBitset, u_targetId)',
            returnType: GLSLDataType.Bool,
            includes: {
              '#include "shared.glsl"': 'assets/shared.glsl',
            },
            textures: [
              {
                uniformName: 'u_testBitset',
                unit: 0,
                texture: bitsetTexture,
              },
            ],
            uniforms: [
              {
                name: 'u_targetId',
                setter: (gl, loc) => gl.uniform1ui(loc, id),
              },
            ],
          });
        };

        // Assert set bits return true
        expect(await checkBit(0)).toBe(true);
        expect(await checkBit(1)).toBe(true);
        expect(await checkBit(31)).toBe(true);
        expect(await checkBit(32)).toBe(true);
        expect(await checkBit(42)).toBe(true);
        expect(await checkBit(65535)).toBe(true);
        expect(await checkBit(65536)).toBe(true);
        expect(await checkBit(70000)).toBe(true);
        expect(await checkBit(100000)).toBe(true);

        // Assert unset bits return false
        expect(await checkBit(2)).toBe(false);
        expect(await checkBit(30)).toBe(false);
        expect(await checkBit(33)).toBe(false);
        expect(await checkBit(43)).toBe(false);
        expect(await checkBit(65534)).toBe(false);
        expect(await checkBit(65537)).toBe(false);
        expect(await checkBit(70001)).toBe(false);
        expect(await checkBit(99999)).toBe(false);
        expect(await checkBit(100001)).toBe(false);
      });
    });

    describe('ViewState UBO integration', () => {
      it('correctly reads selectedLogIndex and viewport values from ViewState block', async () => {
        const gl = tester.gl;
        const uboBuffer = gl.createBuffer()!;
        const source = new ArrayBuffer(32);
        const floatView = new Float32Array(source);
        const uintView = new Uint32Array(source);

        // canvasResolution: vec2(800.0, 600.0) -> offset 0
        floatView[0] = 800.0;
        floatView[1] = 600.0;
        // devicePixelRatio: 2.0 -> offset 8
        floatView[2] = 2.0;
        // pixelsPerMs: 0.5 -> offset 12
        floatView[3] = 0.5;
        // leftEdgeTime: uvec2(1000u, 500u) -> offset 16
        uintView[4] = 1000;
        uintView[5] = 500;
        // selectedLogIndex: 12345u -> offset 24
        uintView[6] = 12345;
        // padding -> offset 28
        uintView[7] = 0;

        gl.bindBuffer(gl.UNIFORM_BUFFER, uboBuffer);
        gl.bufferData(gl.UNIFORM_BUFFER, source, gl.STATIC_DRAW);
        gl.bindBuffer(gl.UNIFORM_BUFFER, null);

        const selectedLogIndex = await tester.runExpression<number>({
          expression: 'vs.selectedLogIndex',
          returnType: GLSLDataType.Uint,
          includes: {
            '#include "shared.glsl"': 'assets/shared.glsl',
          },
          ubos: [
            {
              blockName: 'ViewState',
              bindingPoint: 0,
              buffer: uboBuffer,
            },
          ],
        });
        expect(selectedLogIndex).toBe(12345);

        const pixelsPerMs = await tester.runExpression<number>({
          expression: 'vs.pixelsPerMs',
          returnType: GLSLDataType.Float,
          includes: {
            '#include "shared.glsl"': 'assets/shared.glsl',
          },
          ubos: [
            {
              blockName: 'ViewState',
              bindingPoint: 0,
              buffer: uboBuffer,
            },
          ],
        });
        expect(pixelsPerMs).toBeCloseTo(0.5, 4);

        gl.deleteBuffer(uboBuffer);
      });
    });
  });
});
