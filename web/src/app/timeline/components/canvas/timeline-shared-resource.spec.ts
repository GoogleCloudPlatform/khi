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

import { TimelineRendererSharedResource } from './timeline-shared-resource';
import { SharedTmpBuffer, WebGLUtil } from './glutil';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';
import { BMFontConfig, IconAtlas } from 'src/app/store/domain/style';

describe('TimelineRendererSharedResource', () => {
  let sharedResource: TimelineRendererSharedResource;
  let canvas: HTMLCanvasElement;
  let gl: WebGL2RenderingContext;
  let tmpBuffer: SharedTmpBuffer;

  const mockBMFontNumbers: BMFontConfig = {
    pages: ['assets/zzz-roboto-number-msdf.png'],
    common: {
      lineHeight: 32,
      base: 26,
      scaleW: 256,
      scaleH: 256,
      pages: 1,
      packed: 0,
      alphaChnl: 0,
      redChnl: 0,
      greenChnl: 0,
      blueChnl: 0,
    },
    chars: Array.from({ length: 10 }, (_, i) => ({
      id: 48 + i,
      index: i,
      char: String(i),
      width: 15,
      height: 20,
      xoffset: 1,
      yoffset: 2,
      xadvance: 16,
      chnl: 15,
      x: i * 20,
      y: 0,
      page: 0,
    })),
  };

  beforeEach(async () => {
    canvas = document.createElement('canvas');
    const context = canvas.getContext('webgl2');
    if (!context) {
      throw new Error('WebGL2 context is not supported in this environment');
    }
    gl = context;
    tmpBuffer = new SharedTmpBuffer();
    sharedResource = new TimelineRendererSharedResource();

    // Spy on WebGLUtil static assets loaders to avoid making network requests
    spyOn(WebGLUtil, 'loadTexture').and.callFake(async () => {
      const dummyTex = gl.createTexture();
      if (!dummyTex) {
        throw new Error('Failed to create WebGL texture in mock');
      }
      return dummyTex;
    });

    spyOn(WebGLUtil, 'loadBMFontConfig').and.callFake(async () => {
      return mockBMFontNumbers;
    });
  });

  describe('setup', () => {
    it('should initialize WebGL resources and samplers without styleStore', async () => {
      await sharedResource.setup(gl, tmpBuffer);

      expect(sharedResource.msdfSampler).toBeTruthy();
      expect(sharedResource.uboViewState).toBeTruthy();
      expect(sharedResource.numberMSDFTexture).toBeTruthy();
      expect(sharedResource.uboNumberMSDFParamBuffer).toBeTruthy();

      expect(WebGLUtil.loadTexture).toHaveBeenCalledWith(
        gl,
        'assets/zzz-roboto-number-msdf.png',
      );
      expect(WebGLUtil.loadBMFontConfig).toHaveBeenCalledWith(
        'assets/zzz-roboto-number-msdf.json',
      );
    });
  });

  describe('updateIconAtlas', () => {
    it('should lazily load and configure MSDF icon texture and mappings from styleStore', async () => {
      const mockData = await createMockInspectionData();
      const styleStore = mockData.styleStore;

      // Set up a dummy self-contained IconAtlas in styleStore to avoid network requests
      const nameToCodepoints = new Map<string, string>([
        ['check', String.fromCodePoint(0xe5ca)],
        ['error', String.fromCodePoint(0xe000)],
      ]);

      const dummyCanvas = document.createElement('canvas');
      dummyCanvas.width = 1;
      dummyCanvas.height = 1;

      const dummyBMFontJson: BMFontConfig = {
        pages: ['zzz-material-icons-msdf.png'],
        common: {
          lineHeight: 32,
          base: 26,
          scaleW: 256,
          scaleH: 256,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
        chars: [
          {
            id: 0xe5ca,
            index: 0,
            char: String.fromCodePoint(0xe5ca),
            width: 15,
            height: 20,
            xoffset: 1,
            yoffset: 2,
            xadvance: 16,
            chnl: 15,
            x: 0,
            y: 0,
            page: 0,
          },
          {
            id: 0xe000,
            index: 1,
            char: String.fromCodePoint(0xe000),
            width: 15,
            height: 20,
            xoffset: 1,
            yoffset: 2,
            xadvance: 16,
            chnl: 15,
            x: 20,
            y: 0,
            page: 0,
          },
        ],
      };

      const mockIconAtlas: IconAtlas = {
        msdfIconImage: [dummyCanvas],
        bmfontJson: dummyBMFontJson,
        nameToCodepoints,
      };

      spyOn(styleStore, 'getIconAtlas').and.returnValue(mockIconAtlas);

      // Setup generic resources first
      await sharedResource.setup(gl, tmpBuffer);

      // Perform dynamic icon atlas update
      sharedResource.updateIconAtlas(gl, styleStore);

      expect(sharedResource.iconsMSDFTexture).toBeTruthy();
      expect(sharedResource.iconCodepointMap).toBe(nameToCodepoints);
      expect(sharedResource.bmfontConfigIcons).toBe(dummyBMFontJson);
    });

    it('should avoid uploading icon texture again if the same icon atlas instance is updated', async () => {
      const mockData = await createMockInspectionData();
      const styleStore = mockData.styleStore;

      const nameToCodepoints = new Map<string, string>([
        ['check', String.fromCodePoint(0xe5ca)],
      ]);

      const dummyCanvas = document.createElement('canvas');
      dummyCanvas.width = 1;
      dummyCanvas.height = 1;

      const dummyBMFontJson: BMFontConfig = {
        pages: ['zzz-material-icons-msdf.png'],
        common: {
          lineHeight: 32,
          base: 26,
          scaleW: 256,
          scaleH: 256,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
        chars: [
          {
            id: 0xe5ca,
            index: 0,
            char: String.fromCodePoint(0xe5ca),
            width: 15,
            height: 20,
            xoffset: 1,
            yoffset: 2,
            xadvance: 16,
            chnl: 15,
            x: 0,
            y: 0,
            page: 0,
          },
        ],
      };

      const mockIconAtlas: IconAtlas = {
        msdfIconImage: [dummyCanvas],
        bmfontJson: dummyBMFontJson,
        nameToCodepoints,
      };

      spyOn(styleStore, 'getIconAtlas').and.returnValue(mockIconAtlas);

      await sharedResource.setup(gl, tmpBuffer);

      spyOn(WebGLUtil, 'loadTextureDirect').and.callThrough();

      // First update calls WebGLUtil.loadTextureDirect
      sharedResource.updateIconAtlas(gl, styleStore);
      expect(WebGLUtil.loadTextureDirect).toHaveBeenCalledTimes(1);

      // Second update with same styleStore / icon atlas avoids recreation
      sharedResource.updateIconAtlas(gl, styleStore);
      expect(WebGLUtil.loadTextureDirect).toHaveBeenCalledTimes(1);
    });
  });

  describe('getIconUVSizes', () => {
    it('should return correct texture coordinates and scaling values for a valid icon', async () => {
      const mockData = await createMockInspectionData();
      const styleStore = mockData.styleStore;

      const nameToCodepoints = new Map<string, string>([
        ['check', String.fromCodePoint(0xe5ca)],
      ]);

      const dummyCanvas = document.createElement('canvas');
      dummyCanvas.width = 1;
      dummyCanvas.height = 1;

      const dummyBMFontJson: BMFontConfig = {
        pages: ['zzz-material-icons-msdf.png'],
        common: {
          lineHeight: 32,
          base: 26,
          scaleW: 256,
          scaleH: 256,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
        chars: [
          {
            id: 0xe5ca,
            index: 0,
            char: String.fromCodePoint(0xe5ca),
            width: 16,
            height: 16,
            xoffset: 2,
            yoffset: 2,
            xadvance: 18,
            chnl: 15,
            x: 16,
            y: 32,
            page: 0,
          },
        ],
      };

      const mockIconAtlas: IconAtlas = {
        msdfIconImage: [dummyCanvas],
        bmfontJson: dummyBMFontJson,
        nameToCodepoints,
      };

      spyOn(styleStore, 'getIconAtlas').and.returnValue(mockIconAtlas);

      await sharedResource.setup(gl, tmpBuffer);
      sharedResource.updateIconAtlas(gl, styleStore);

      const uvSizes = sharedResource.getIconUVSizes('check');

      // u = x / scaleW = 16 / 256 = 0.0625
      // v = y / scaleH = 32 / 256 = 0.125
      // widthRatio = width / scaleW = 16 / 256 = 0.0625
      // heightRatio = height / scaleH = 16 / 256 = 0.0625
      expect(uvSizes[0]).toBeCloseTo(0.0625, 5);
      expect(uvSizes[1]).toBeCloseTo(0.125, 5);
      expect(uvSizes[2]).toBeCloseTo(0.0625, 5);
      expect(uvSizes[3]).toBeCloseTo(0.0625, 5);
    });

    it('should throw an error if requested icon name is not registered in codepoints', async () => {
      const mockData = await createMockInspectionData();
      const styleStore = mockData.styleStore;

      const nameToCodepoints = new Map<string, string>();

      const dummyCanvas = document.createElement('canvas');
      dummyCanvas.width = 1;
      dummyCanvas.height = 1;

      const dummyBMFontJson: BMFontConfig = {
        pages: ['zzz-material-icons-msdf.png'],
        common: {
          lineHeight: 32,
          base: 26,
          scaleW: 256,
          scaleH: 256,
          pages: 1,
          packed: 0,
          alphaChnl: 0,
          redChnl: 0,
          greenChnl: 0,
          blueChnl: 0,
        },
        chars: [],
      };

      const mockIconAtlas: IconAtlas = {
        msdfIconImage: [dummyCanvas],
        bmfontJson: dummyBMFontJson,
        nameToCodepoints,
      };

      spyOn(styleStore, 'getIconAtlas').and.returnValue(mockIconAtlas);

      await sharedResource.setup(gl, tmpBuffer);
      sharedResource.updateIconAtlas(gl, styleStore);

      expect(() => {
        sharedResource.getIconUVSizes('invalid-icon');
      }).toThrowError('icon code invalid-icon is not found');
    });
  });
});
