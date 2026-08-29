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
  GLSLIncludeReplace,
  WebGLUtil,
} from 'src/app/timeline/components/canvas/glutil';
import { WebGLContextLostException } from 'src/app/timeline/components/canvas/glcontextmanager';

/**
 * Supported GLSL data types for evaluated expressions and output varyings.
 */
export enum GLSLDataType {
  Bool = 'bool',
  Float = 'float',
  Int = 'int',
  Uint = 'uint',
  Vec2 = 'vec2',
  Vec3 = 'vec3',
  Vec4 = 'vec4',
  IVec2 = 'ivec2',
  IVec3 = 'ivec3',
  IVec4 = 'ivec4',
  UVec2 = 'uvec2',
  UVec3 = 'uvec3',
  UVec4 = 'uvec4',
}

/**
 * Sampler type for GLSL textures.
 */
export enum GLSLSamplerType {
  Usampler2D = 'usampler2D',
  Sampler2D = 'sampler2D',
  Isampler2D = 'isampler2D',
}

/**
 * Texture binding configuration for a GLSL test run.
 */
export interface GLSLTextureBinding {
  /** The uniform name for the sampler in the shader. */
  readonly uniformName: string;
  /** The WebGL texture unit index to bind (e.g. 0, 1, 2). */
  readonly unit: number;
  /** The WebGL texture instance to bind. */
  readonly texture: WebGLTexture;
  /** The GLSL sampler type declaration. Defaults to usampler2D. */
  readonly samplerType?: GLSLSamplerType;
}

/**
 * Uniform setter configuration for custom uniform values.
 */
export interface GLSLUniformBinding {
  /** The name of the uniform in the shader. */
  readonly name: string;
  /** Callback to set the uniform value on the WebGL context. */
  readonly setter: (
    gl: WebGL2RenderingContext,
    location: WebGLUniformLocation,
  ) => void;
}

/**
 * Uniform Buffer Object binding configuration for a GLSL test run.
 */
export interface GLSLUBOBinding {
  /** The name of the uniform block in the shader. */
  readonly blockName: string;
  /** The binding point index to assign to the block. */
  readonly bindingPoint: number;
  /** The WebGL buffer containing the UBO data. */
  readonly buffer: WebGLBuffer;
}

/**
 * Options for evaluating a single GLSL expression or function call.
 */
export interface GLSLExpressionOptions {
  /** The GLSL expression to evaluate (e.g. `checkBitset(u_bitset, u_id)`). */
  readonly expression: string;
  /** The expected return data type of the expression. */
  readonly returnType: GLSLDataType;
  /** Optional header declarations (functions, structs, constants) or includes. */
  readonly declarations?: string;
  /** Map of tokens to file paths for GLSL inclusion/replacement (e.g. `#include "shared.glsl"`). */
  readonly includes?: GLSLIncludeReplace;
  /** Optional texture bindings. */
  readonly textures?: readonly GLSLTextureBinding[];
  /** Optional uniform setters. */
  readonly uniforms?: readonly GLSLUniformBinding[];
  /** Optional UBO bindings. */
  readonly ubos?: readonly GLSLUBOBinding[];
}

/**
 * Options for executing a custom vertex shader source with transform feedback.
 */
export interface GLSLShaderOptions {
  /** Full vertex shader source code. */
  readonly vertexShaderSource: string;
  /** The name of the output varying capturing the result. */
  readonly outputVarying: string;
  /** The data type of the output varying. */
  readonly returnType: GLSLDataType;
  /** Map of tokens to file paths for GLSL inclusion/replacement. */
  readonly includes?: GLSLIncludeReplace;
  /** Optional texture bindings. */
  readonly textures?: readonly GLSLTextureBinding[];
  /** Optional uniform setters. */
  readonly uniforms?: readonly GLSLUniformBinding[];
  /** Optional UBO bindings. */
  readonly ubos?: readonly GLSLUBOBinding[];
}

/**
 * Test utility that compiles and executes GLSL shader functions and expressions in WebGL2
 * using Transform Feedback for exact typed value readback.
 */
export class GLSLFunctionTester {
  private canvas?: HTMLCanvasElement;
  private glContext?: WebGL2RenderingContext;

  /**
   * Gets the underlying WebGL2 rendering context.
   */
  public get gl(): WebGL2RenderingContext {
    if (!this.glContext) {
      throw new Error(
        'GLSLFunctionTester is not initialized. Call setup() first.',
      );
    }
    return this.glContext;
  }

  /**
   * Initializes an offscreen canvas and WebGL2 context for shader testing.
   */
  public setup(): void {
    this.canvas = document.createElement('canvas');
    this.canvas.width = 1;
    this.canvas.height = 1;
    const gl = this.canvas.getContext('webgl2');
    if (!gl) {
      throw new WebGLContextLostException(
        'WebGL2 is not supported in this test environment.',
      );
    }
    this.glContext = gl;
  }

  /**
   * Disposes the WebGL2 context and resources.
   */
  public dispose(): void {
    if (this.glContext) {
      const ext = this.glContext.getExtension('WEBGL_lose_context');
      if (ext) {
        ext.loseContext();
      }
      this.glContext = undefined;
    }
    if (this.canvas) {
      this.canvas.remove();
      this.canvas = undefined;
    }
  }

  /**
   * Creates an R32UI 2D texture from raw 32-bit bitset words.
   *
   * @param words The 32-bit words array representing the bitset.
   * @param width The width of the texture in texels (defaults to 2048).
   * @returns The initialized WebGL texture.
   */
  public createBitsetTexture(
    words: Uint32Array,
    width: number = 2048,
  ): WebGLTexture {
    const gl = this.gl;
    const height = Math.max(1, Math.ceil(words.length / width));
    const padded = new Uint32Array(width * height);
    padded.set(words);

    const texture = gl.createTexture();
    if (!texture) {
      throw new WebGLContextLostException('Failed to create bitset texture');
    }
    gl.bindTexture(gl.TEXTURE_2D, texture);
    gl.texImage2D(
      gl.TEXTURE_2D,
      0,
      gl.R32UI,
      width,
      height,
      0,
      gl.RED_INTEGER,
      gl.UNSIGNED_INT,
      padded,
    );
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    gl.bindTexture(gl.TEXTURE_2D, null);
    return texture;
  }

  /**
   * Evaluates a GLSL expression and returns the typed result.
   *
   * @param options Expression evaluation configuration.
   * @returns A promise resolving to the evaluated value (boolean, number, or number[]).
   */
  public async runExpression<T>(options: GLSLExpressionOptions): Promise<T> {
    const includeTokens = Object.keys(options.includes ?? {}).join('\n');
    const declarations = [includeTokens, options.declarations ?? '']
      .filter(Boolean)
      .join('\n');
    const textureUniforms = (options.textures ?? [])
      .map(
        (t) =>
          `uniform highp ${t.samplerType ?? GLSLSamplerType.Usampler2D} ${t.uniformName};`,
      )
      .join('\n');

    const varyingType =
      options.returnType === GLSLDataType.Bool
        ? GLSLDataType.Uint
        : options.returnType;

    const assignment =
      options.returnType === GLSLDataType.Bool
        ? `out_varying_result = (${options.expression}) ? 1u : 0u;`
        : `out_varying_result = ${options.expression};`;

    const isIntegerType =
      options.returnType === GLSLDataType.Bool ||
      options.returnType === GLSLDataType.Int ||
      options.returnType === GLSLDataType.Uint ||
      options.returnType === GLSLDataType.IVec2 ||
      options.returnType === GLSLDataType.IVec3 ||
      options.returnType === GLSLDataType.IVec4 ||
      options.returnType === GLSLDataType.UVec2 ||
      options.returnType === GLSLDataType.UVec3 ||
      options.returnType === GLSLDataType.UVec4;

    const qualifier = isIntegerType ? 'flat out' : 'out';

    const vertexShaderSource = `
#version 300 es
precision highp float;
precision highp int;

${declarations}
${textureUniforms}

${qualifier} ${varyingType} out_varying_result;

void main() {
  ${assignment}
}
`;

    return this.runShader<T>({
      vertexShaderSource,
      outputVarying: 'out_varying_result',
      returnType: options.returnType,
      includes: options.includes,
      textures: options.textures,
      uniforms: options.uniforms,
      ubos: options.ubos,
    });
  }

  /**
   * Executes a custom vertex shader with transform feedback and reads back the output varying.
   *
   * @param options Shader execution configuration.
   * @returns A promise resolving to the varying result value.
   */
  public async runShader<T>(options: GLSLShaderOptions): Promise<T> {
    const gl = this.gl;

    let vss = options.vertexShaderSource;
    if (options.includes) {
      for (const [key, value] of Object.entries(options.includes)) {
        const fileContent = await WebGLUtil.getShaderString(value);
        vss = vss.replaceAll(key, fileContent);
      }
    }
    vss = vss.trimStart();

    const vs = gl.createShader(gl.VERTEX_SHADER);
    if (!vs) {
      throw new WebGLContextLostException('Failed to create vertex shader');
    }
    gl.shaderSource(vs, vss);
    gl.compileShader(vs);
    if (!gl.getShaderParameter(vs, gl.COMPILE_STATUS)) {
      const log = gl.getShaderInfoLog(vs);
      gl.deleteShader(vs);
      throw new Error(
        `GLSL compilation error:\n${log}\n\nShader Source:\n${vss}`,
      );
    }

    const fs = gl.createShader(gl.FRAGMENT_SHADER);
    if (!fs) {
      gl.deleteShader(vs);
      throw new WebGLContextLostException('Failed to create fragment shader');
    }
    gl.shaderSource(
      fs,
      '#version 300 es\nprecision highp float;\nvoid main() {}\n',
    );
    gl.compileShader(fs);

    const program = gl.createProgram();
    if (!program) {
      gl.deleteShader(vs);
      gl.deleteShader(fs);
      throw new WebGLContextLostException('Failed to create WebGL program');
    }
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);

    gl.transformFeedbackVaryings(
      program,
      [options.outputVarying],
      gl.SEPARATE_ATTRIBS,
    );

    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      const log = gl.getProgramInfoLog(program);
      gl.deleteProgram(program);
      gl.deleteShader(vs);
      gl.deleteShader(fs);
      throw new Error(`GLSL linking error:\n${log}`);
    }

    const dummyBuffers: WebGLBuffer[] = [];
    const activeUniformBlocks = gl.getProgramParameter(
      program,
      gl.ACTIVE_UNIFORM_BLOCKS,
    ) as number;
    const boundPoints = new Set<number>();

    if (options.ubos) {
      for (const ubo of options.ubos) {
        WebGLUtil.setProgramUniformBlockBinding(
          gl,
          program,
          ubo.blockName,
          ubo.bindingPoint,
        );
        gl.bindBufferBase(gl.UNIFORM_BUFFER, ubo.bindingPoint, ubo.buffer);
        boundPoints.add(ubo.bindingPoint);
      }
    }

    let nextBindingPoint = 0;
    for (let i = 0; i < activeUniformBlocks; i++) {
      const blockName = gl.getActiveUniformBlockName(program, i);
      const isBound = options.ubos?.some((u) => u.blockName === blockName);
      if (!isBound && blockName) {
        while (boundPoints.has(nextBindingPoint)) {
          nextBindingPoint++;
        }
        const dummyBuffer = gl.createBuffer();
        if (dummyBuffer) {
          dummyBuffers.push(dummyBuffer);
          const blockSize = gl.getActiveUniformBlockParameter(
            program,
            i,
            gl.UNIFORM_BLOCK_DATA_SIZE,
          ) as number;
          gl.bindBuffer(gl.UNIFORM_BUFFER, dummyBuffer);
          gl.bufferData(
            gl.UNIFORM_BUFFER,
            Math.max(blockSize, 256),
            gl.DYNAMIC_DRAW,
          );
          gl.uniformBlockBinding(program, i, nextBindingPoint);
          gl.bindBufferBase(gl.UNIFORM_BUFFER, nextBindingPoint, dummyBuffer);
          boundPoints.add(nextBindingPoint);
        }
      }
    }

    const typeConfig = this.getTypeConfig(options.returnType);
    const feedbackBuffer = gl.createBuffer();
    if (!feedbackBuffer) {
      gl.deleteProgram(program);
      gl.deleteShader(vs);
      gl.deleteShader(fs);
      throw new WebGLContextLostException('Failed to create feedback buffer');
    }
    gl.bindBuffer(gl.TRANSFORM_FEEDBACK_BUFFER, feedbackBuffer);
    gl.bufferData(
      gl.TRANSFORM_FEEDBACK_BUFFER,
      typeConfig.byteSize,
      gl.STATIC_READ,
    );

    const transformFeedback = gl.createTransformFeedback();
    if (!transformFeedback) {
      gl.deleteBuffer(feedbackBuffer);
      gl.deleteProgram(program);
      gl.deleteShader(vs);
      gl.deleteShader(fs);
      throw new WebGLContextLostException(
        'Failed to create transform feedback object',
      );
    }
    gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, transformFeedback);
    gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, feedbackBuffer);

    gl.useProgram(program);

    if (options.textures) {
      for (const t of options.textures) {
        gl.activeTexture(gl.TEXTURE0 + t.unit);
        gl.bindTexture(gl.TEXTURE_2D, t.texture);
        gl.bindSampler(t.unit, null);
        const loc = gl.getUniformLocation(program, t.uniformName);
        if (loc !== null) {
          gl.uniform1i(loc, t.unit);
        }
      }
    }

    if (options.uniforms) {
      for (const u of options.uniforms) {
        const loc = gl.getUniformLocation(program, u.name);
        if (loc !== null) {
          u.setter(gl, loc);
        }
      }
    }

    gl.enable(gl.RASTERIZER_DISCARD);
    gl.beginTransformFeedback(gl.POINTS);
    gl.drawArrays(gl.POINTS, 0, 1);
    gl.endTransformFeedback();
    gl.disable(gl.RASTERIZER_DISCARD);

    gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, null);
    gl.bindBuffer(gl.TRANSFORM_FEEDBACK_BUFFER, feedbackBuffer);

    const readArray = typeConfig.createArray();
    gl.getBufferSubData(gl.TRANSFORM_FEEDBACK_BUFFER, 0, readArray);

    gl.bindBuffer(gl.TRANSFORM_FEEDBACK_BUFFER, null);
    dummyBuffers.forEach((b) => gl.deleteBuffer(b));
    gl.deleteTransformFeedback(transformFeedback);
    gl.deleteBuffer(feedbackBuffer);
    gl.deleteProgram(program);
    gl.deleteShader(vs);
    gl.deleteShader(fs);

    return typeConfig.formatResult(readArray) as T;
  }

  private getTypeConfig(type: GLSLDataType): {
    byteSize: number;
    createArray: () => ArrayBufferView;
    formatResult: (arr: ArrayBufferView) => unknown;
  } {
    switch (type) {
      case GLSLDataType.Bool:
        return {
          byteSize: 4,
          createArray: () => new Uint32Array(1),
          formatResult: (arr) => (arr as Uint32Array)[0] !== 0,
        };
      case GLSLDataType.Uint:
        return {
          byteSize: 4,
          createArray: () => new Uint32Array(1),
          formatResult: (arr) => (arr as Uint32Array)[0],
        };
      case GLSLDataType.Int:
        return {
          byteSize: 4,
          createArray: () => new Int32Array(1),
          formatResult: (arr) => (arr as Int32Array)[0],
        };
      case GLSLDataType.Float:
        return {
          byteSize: 4,
          createArray: () => new Float32Array(1),
          formatResult: (arr) => (arr as Float32Array)[0],
        };
      case GLSLDataType.Vec2:
        return {
          byteSize: 8,
          createArray: () => new Float32Array(2),
          formatResult: (arr) => Array.from(arr as Float32Array),
        };
      case GLSLDataType.Vec3:
        return {
          byteSize: 12,
          createArray: () => new Float32Array(3),
          formatResult: (arr) => Array.from(arr as Float32Array),
        };
      case GLSLDataType.Vec4:
        return {
          byteSize: 16,
          createArray: () => new Float32Array(4),
          formatResult: (arr) => Array.from(arr as Float32Array),
        };
      case GLSLDataType.IVec2:
        return {
          byteSize: 8,
          createArray: () => new Int32Array(2),
          formatResult: (arr) => Array.from(arr as Int32Array),
        };
      case GLSLDataType.IVec3:
        return {
          byteSize: 12,
          createArray: () => new Int32Array(3),
          formatResult: (arr) => Array.from(arr as Int32Array),
        };
      case GLSLDataType.IVec4:
        return {
          byteSize: 16,
          createArray: () => new Int32Array(4),
          formatResult: (arr) => Array.from(arr as Int32Array),
        };
      case GLSLDataType.UVec2:
        return {
          byteSize: 8,
          createArray: () => new Uint32Array(2),
          formatResult: (arr) => Array.from(arr as Uint32Array),
        };
      case GLSLDataType.UVec3:
        return {
          byteSize: 12,
          createArray: () => new Uint32Array(3),
          formatResult: (arr) => Array.from(arr as Uint32Array),
        };
      case GLSLDataType.UVec4:
        return {
          byteSize: 16,
          createArray: () => new Uint32Array(4),
          formatResult: (arr) => Array.from(arr as Uint32Array),
        };
    }
  }
}
