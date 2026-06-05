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

import { StyleStore } from 'src/app/store/domain/style-store';
import { RendererConvertUtil } from './convertutil';
import { WebGLContextLostException } from './glcontextmanager';
import { SharedTmpBuffer, WebGLUtil } from './glutil';
import {
  BMFontChar,
  BMFontConfig,
  IconAtlas,
} from 'src/app/store/domain/style';

/**
 * Represents the state required by shared timeline rendering resources.
 * This includes viewport dimensions, zoom level, and time-related parameters.
 */
export interface TimelineRendererSharedResourceState {
  /** The width of the viewport in logical pixels. */
  width: number;
  /** The height of the viewport in logical pixels. */
  height: number;
  /** The device pixel ratio (DPR) for handling high-DPI displays. */
  devicePixelRatio: number;
  /** The current zoom level in pixels per millisecond. */
  pixelsPerMs: number;
  /** The time (in unix milliseconds) at the left edge of the viewport. */
  leftEdgeTime: number;
}

/**
 * Shared resources for timeline rendering, such as fonts, textures, and common uniform buffers.
 * This class manages MSDF textures for text and icons, and updates the shared view state UBO.
 */
export class TimelineRendererSharedResource {
  public readonly MAX_NUMBER_FONTS = 10;
  public readonly MAX_ICON_FONTS = 128;

  /** Uniform Buffer Object for storing the view state (viewport, time). */
  uboViewState!: WebGLBuffer;
  uboViewStateSource!: ArrayBuffer;

  /** Uniform Buffer Object for storing parameters related to the number MSDF font atlas. */
  uboNumberMSDFParamBuffer!: WebGLBuffer;
  /** Texture containing the MSDF atlas for number glyphs. */
  numberMSDFTexture!: WebGLTexture;
  /** Sampler for MSDF textures. */
  msdfSampler!: WebGLSampler;

  /** Texture containing the MSDF atlas for Material Icons. */
  iconsMSDFTexture!: WebGLTexture;
  /** Sampler for the icon MSDF texture. */
  iconsMSDFSampler!: WebGLSampler;

  /** Configuration for the icon MSDF font. */
  bmfontConfigIcons!: BMFontConfig;
  /** Map of icon names to their Unicode codepoints. */
  iconCodepointMap!: Map<string, string>;

  private lastIconAtlas?: IconAtlas;

  /**
   * Initializes the shared resources.
   * Loads textures, font configurations, and creates necessary buffers and samplers.
   *
   * @param gl The WebGL2 rendering context.
   */
  async setup(gl: WebGL2RenderingContext, tmpBuffer: SharedTmpBuffer) {
    this.msdfSampler = gl.createSampler();
    if (this.msdfSampler === null) {
      throw new WebGLContextLostException('Failed to create sampler');
    }
    gl.samplerParameteri(this.msdfSampler, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.samplerParameteri(this.msdfSampler, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.samplerParameteri(this.msdfSampler, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.samplerParameteri(this.msdfSampler, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);

    this.uboViewState = gl.createBuffer();
    if (this.uboViewState === null) {
      throw new WebGLContextLostException('Failed to create buffer');
    }
    this.uboViewStateSource = new ArrayBuffer(32); // 24 + 8 padding
    gl.bindBuffer(gl.UNIFORM_BUFFER, this.uboViewState);
    gl.bufferData(gl.UNIFORM_BUFFER, this.uboViewStateSource, gl.DYNAMIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);

    this.numberMSDFTexture = await WebGLUtil.loadTexture(
      gl,
      'assets/zzz-roboto-number-msdf.png',
    );

    const bmfontConfigNumbers = await WebGLUtil.loadBMFontConfig(
      'assets/zzz-roboto-number-msdf.json',
    );
    const numberChars = new Array<BMFontChar>(10);
    for (let i = 0; i < 10; i++) {
      const char = bmfontConfigNumbers.chars[i];
      numberChars[+char.char] = char;
    }

    const dv = tmpBuffer.dataView(4 * 8 * this.MAX_NUMBER_FONTS);
    for (let i = 0; i < this.MAX_NUMBER_FONTS; i++) {
      const char = numberChars[i];
      dv.setFloat32(
        i * 16 + 0,
        char.x / bmfontConfigNumbers.common.scaleW,
        true,
      );
      dv.setFloat32(
        i * 16 + 4,
        char.y / bmfontConfigNumbers.common.scaleH,
        true,
      );
      dv.setFloat32(
        i * 16 + 8,
        char.width / bmfontConfigNumbers.common.scaleW,
        true,
      );
      dv.setFloat32(
        i * 16 + 12,
        char.height / bmfontConfigNumbers.common.scaleH,
        true,
      );
    }
    const offset = 4 * 4 * this.MAX_NUMBER_FONTS;
    for (let i = 0; i < this.MAX_NUMBER_FONTS; i++) {
      const char = numberChars[i];
      dv.setFloat32(
        offset + i * 16 + 0,
        char.xoffset / bmfontConfigNumbers.common.scaleW,
        true,
      );
      dv.setFloat32(
        offset + i * 16 + 4,
        char.yoffset / bmfontConfigNumbers.common.scaleH,
        true,
      );
      dv.setFloat32(offset + i * 16 + 8, char.xadvance, true);
      // 4 byte padding for std140
    }
    this.uboNumberMSDFParamBuffer = gl.createBuffer();
    gl.bindBuffer(gl.UNIFORM_BUFFER, this.uboNumberMSDFParamBuffer);
    gl.bufferData(gl.UNIFORM_BUFFER, dv, gl.STATIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);
  }

  /**
   * Dynamically updates the icon atlas from the style store if it has changed.
   * Re-allocates/uploads the icons texture when the icon atlas is updated.
   *
   * @param gl The WebGL2 rendering context.
   * @param styleStore The style store containing the icon atlas.
   */
  updateIconAtlas(gl: WebGL2RenderingContext, styleStore: StyleStore) {
    const iconAtlas = styleStore.getIconAtlas();
    if (!iconAtlas) {
      // If no inspection data has been loaded yet, there will be no icon atlas.
      return;
    }

    if (this.lastIconAtlas === iconAtlas) {
      return;
    }
    this.lastIconAtlas = iconAtlas;

    if (iconAtlas.msdfIconImage.length === 0) {
      return;
    }

    if (iconAtlas.msdfIconImage.length > 1) {
      // TODO: support multiple msdf icon atlas textures to support large number of icon varieties support.
      throw new Error('Multiple msdf icon images are not yet supported');
    }

    if (this.iconsMSDFTexture) {
      gl.deleteTexture(this.iconsMSDFTexture);
    }

    this.iconsMSDFTexture = WebGLUtil.loadTextureDirect(
      gl,
      iconAtlas.msdfIconImage[0],
    );
    this.iconCodepointMap = iconAtlas.nameToCodepoints;
    this.bmfontConfigIcons = iconAtlas.bmfontJson;
  }

  /**
   * Retrieves the UV coordinates and size ratios for a given icon.
   *
   * @param iconName The name of the icon (e.g., "check").
   * @returns A tuple containing [u, v, widthRatio, heightRatio].
   * @throws Error if the icon config is not loaded or the icon is not found.
   */
  getIconUVSizes(iconName: string): [number, number, number, number] {
    if (this.bmfontConfigIcons === undefined)
      throw new Error('icon bmfont config file is not yet loaded');
    const iconCodePoint = this.iconCodepointMap.get(iconName);
    if (!iconCodePoint) {
      throw new Error(`icon code ${iconName} is not found`);
    }
    for (let i = 0; i < this.bmfontConfigIcons.chars.length; i++) {
      const char = this.bmfontConfigIcons.chars[i];
      if (char.char === iconCodePoint) {
        const scaleW = this.bmfontConfigIcons.common.scaleW;
        const scaleH = this.bmfontConfigIcons.common.scaleH;
        return [
          char.x / scaleW,
          char.y / scaleH,
          char.width / scaleW,
          char.height / scaleH,
        ];
      }
    }
    throw new Error(`icon code ${iconName} is not found`);
  }

  /**
   * Updates the shared view state UBO with the latest viewport and time information.
   * This should be called before rendering any frame.
   *
   * @param gl The WebGL2 rendering context.
   * @param state The current state of the timeline renderer (viewport, zoom, etc.).
   */
  beforeRender(
    gl: WebGL2RenderingContext,
    state: TimelineRendererSharedResourceState,
  ) {
    const dv = new DataView(this.uboViewStateSource);
    dv.setFloat32(0, state.width, true);
    dv.setFloat32(4, state.height, true);
    dv.setFloat32(8, state.devicePixelRatio, true);
    dv.setFloat32(12, state.pixelsPerMs, true);
    const [seconds, nanoSeconds] =
      RendererConvertUtil.splitTimeToSecondsAndNanoSeconds(state.leftEdgeTime);
    dv.setUint32(16, seconds, true);
    dv.setUint32(20, nanoSeconds, true);
    dv.setUint32(24, 0, true); // these are paddings for std140
    dv.setUint32(28, 0, true);
    gl.bindBuffer(gl.UNIFORM_BUFFER, this.uboViewState);
    gl.bufferData(gl.UNIFORM_BUFFER, this.uboViewStateSource, gl.DYNAMIC_DRAW);
    gl.bindBuffer(gl.UNIFORM_BUFFER, null);
  }
}
