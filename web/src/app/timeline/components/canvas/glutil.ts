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

import { WebGLContextLostException } from './glcontextmanager';

/**
 * BMFontConfig is the JSON representation of a BMFont config.
 * See https://github.com/Experience-Monks/load-bmfont/blob/master/json-spec.md
 */
export interface BMFontConfig {
  pages: string[];
  chars: BMFontChar[];
  common: BMFontCommon;
}

/**
 * Represents a single character in the BMFont configuration.
 */
export interface BMFontChar {
  id: number;
  index: number;
  char: string;
  width: number;
  height: number;
  xoffset: number;
  yoffset: number;
  xadvance: number;
  chnl: number;
  x: number;
  y: number;
  page: number;
}

/**
 * Represents common properties in the BMFont configuration.
 */
export interface BMFontCommon {
  lineHeight: number;
  base: number;
  scaleW: number;
  scaleH: number;
  pages: number;
  packed: number;
  alphaChnl: number;
  redChnl: number;
  greenChnl: number;
  blueChnl: number;
}

/**
 * Map of string tokens to be replaced in GLSL shader source code.
 * Key is the token to replace, Value is the file path to the replacement content.
 */
export type GLSLIncludeReplace = { [replaceToken: string]: string };

/**
 * Utility class for WebGL providing multiple static methods for miscellaneous WebGL tasks.
 */
export class WebGLUtil {
  /**
   * Compiles and links vertex and fragment shaders into a WebGL program.
   * Supports simple string replacement for including other GLSL files.
   *
   * @param gl The WebGL2 rendering context.
   * @param vertexShaderPath Path to the vertex shader file.
   * @param fragmentShaderPath Path to the fragment shader file.
   * @param glslIncludeReplaceFilePaths Map of tokens to file paths for GLSL inclusion/replacement.
   * @returns A promise that resolves to the linked WebGLProgram.
   */
  public static async compileAndLinkShaders(
    gl: WebGL2RenderingContext,
    vertexShaderPath: string,
    fragmentShaderPath: string,
    glslIncludeReplaceFilePaths: GLSLIncludeReplace = {},
  ): Promise<WebGLProgram> {
    const includes: GLSLIncludeReplace = {};
    for (const [key, value] of Object.entries(glslIncludeReplaceFilePaths)) {
      includes[key] = await this.getShaderString(value);
    }
    let vss = await this.getShaderString(vertexShaderPath);
    let fss = await this.getShaderString(fragmentShaderPath);

    for (const [key, value] of Object.entries(includes)) {
      vss = vss.replaceAll(key, value);
      fss = fss.replaceAll(key, value);
    }

    const vs = this.createAndCompileShader(gl, vss, gl.VERTEX_SHADER);
    const fs = this.createAndCompileShader(gl, fss, gl.FRAGMENT_SHADER);

    const program = gl.createProgram();
    if (program === null) {
      throw new WebGLContextLostException('Failed to create program');
    }
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      console.error(
        `Linking error on ${vertexShaderPath} and ${fragmentShaderPath}`,
      );
      console.error(gl.getProgramInfoLog(program));
    }
    return program;
  }

  /**
   * Loads an image from the specified path and creates a WebGL texture from it.
   *
   * @param gl The WebGL2 rendering context.
   * @param texturePath Path to the image file.
   * @returns A promise that resolves to the created WebGLTexture.
   * @throws WebGLContextLostException if texture creation fails.
   */
  public static async loadTexture(
    gl: WebGL2RenderingContext,
    texturePath: string,
  ): Promise<WebGLTexture> {
    const image = await this.loadImage(texturePath);
    const texture = gl.createTexture();
    if (texture === null)
      throw new WebGLContextLostException('Failed to create texture');
    gl.bindTexture(gl.TEXTURE_2D, texture);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, gl.RGBA, gl.UNSIGNED_BYTE, image);
    gl.bindTexture(gl.TEXTURE_2D, null);
    return texture;
  }

  /**
   * Loads a BMFont configuration JSON file.
   *
   * @param path Path to the BMFont JSON file.
   * @returns A promise that resolves to the BMFontConfig.
   */
  public static async loadBMFontConfig(path: string): Promise<BMFontConfig> {
    const result = await fetch(path);
    return result.json();
  }

  /**
   * Sets the binding point for a uniform block in a WebGL program.
   *
   * @param gl The WebGL2 rendering context.
   * @param program The WebGL program.
   * @param uniformBlockName The name of the uniform block.
   * @param uniformBlockBinding The binding point index.
   * @throws Error if the uniform block is not found.
   */
  public static setProgramUniformBlockBinding(
    gl: WebGL2RenderingContext,
    program: WebGLProgram,
    uniformBlockName: string,
    uniformBlockBinding: number,
  ) {
    const uniformBlockIndex = gl.getUniformBlockIndex(
      program,
      uniformBlockName,
    );
    if (uniformBlockIndex === -1) {
      throw new Error(`Uniform block ${uniformBlockName} not found`);
    }
    gl.uniformBlockBinding(program, uniformBlockIndex, uniformBlockBinding);
  }

  private static async loadImage(imagePath: string): Promise<HTMLImageElement> {
    const image = new Image();
    image.src = imagePath;
    await image.decode();
    return image;
  }

  private static createAndCompileShader(
    gl: WebGL2RenderingContext,
    shaderSource: string,
    shaderType: number,
  ): WebGLShader {
    const shader = gl.createShader(shaderType);
    if (shader === null)
      throw new WebGLContextLostException('Failed to create shader');
    gl.shaderSource(shader, shaderSource);
    gl.compileShader(shader);
    if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
      console.error(`Compilation error\n${shaderSource}`);
      console.error(gl.getShaderInfoLog(shader));
    }
    return shader;
  }

  private static async getShaderString(path: string): Promise<string> {
    const result = await fetch(path);
    return result.text();
  }
}
