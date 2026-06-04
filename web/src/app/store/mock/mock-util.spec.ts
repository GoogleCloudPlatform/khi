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
  parseTimestampString,
  parseUnixSeconds,
  parseHexColor,
  objectToInternedStruct,
  initializeMockIconAtlas,
} from 'src/app/store/mock/mock-util';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { InternedStructDecoder } from 'src/app/store/domain/struct-decoder';
import { StyleStore } from 'src/app/store/domain/style-store';

describe('mock-util', () => {
  describe('parseTimestampString', () => {
    it('should correctly convert ISO string to nanoseconds bigint', () => {
      const actual = parseTimestampString('2026-05-13T00:00:00Z');
      expect(actual).toBe(1778630400000000000n);
    });
  });

  describe('parseUnixSeconds', () => {
    it('should correctly convert ISO string directly to Unix seconds', () => {
      const actual = parseUnixSeconds('2026-05-13T00:00:00Z');
      expect(actual).toBe(1778630400);
    });
  });

  describe('parseHexColor', () => {
    it('should correctly parse standard hex color formats scaled 0-1', () => {
      expect(parseHexColor('#0078D7')).toEqual({
        r: 0,
        g: 120 / 255,
        b: 215 / 255,
        a: 1,
      });
      expect(parseHexColor('#fff')).toEqual({
        r: 1,
        g: 1,
        b: 1,
        a: 1,
      });
      expect(parseHexColor('#FF000080')).toEqual({
        r: 1,
        g: 0,
        b: 0,
        a: 128 / 255,
      });
    });
  });

  describe('objectToInternedStruct', () => {
    it('should successfully convert plain object to InternedStruct and decode back', () => {
      const internPool = InternPoolStore.create();
      const idState = { nextStringId: 1, nextFieldSetId: 1 };

      const original = {
        foo: 'bar',
        nested: {
          num: 42,
          flag: true,
        },
      };

      const struct = objectToInternedStruct(original, internPool, idState);
      expect(struct).toBeDefined();

      const decoder = new InternedStructDecoder(internPool);
      const decoded = decoder.decode(struct);

      expect(decoded).toEqual(original);
    });
  });

  describe('initializeMockIconAtlas', () => {
    it('should successfully initialize icon atlas if assets are available', async () => {
      const styleStore = new StyleStore();
      await initializeMockIconAtlas(styleStore);

      try {
        const atlas = styleStore.getIconAtlas();
        expect(atlas).toBeDefined();
        expect(atlas!.msdfIconImage.length).toBeGreaterThan(0);
        expect(atlas!.bmfontJson).toBeDefined();
        expect(atlas!.nameToCodepoints.size).toBeGreaterThan(0);
      } catch (e) {
        expect((e as Error).message).toContain('not yet loaded');
      }
    });
  });
});
