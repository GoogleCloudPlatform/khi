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

import { resolveUseBinaryFormat } from 'src/app/services/api/transport-config-resolver';
import { parseBooleanQueryParam } from 'src/app/utils/config-resolver';
import { environment } from 'src/environments/environment';

describe('transport-config-resolver', () => {
  describe('parseBooleanQueryParam', () => {
    it('parses truthy boolean strings correctly', () => {
      expect(parseBooleanQueryParam('true')).toBeTrue();
      expect(parseBooleanQueryParam('TRUE')).toBeTrue();
      expect(parseBooleanQueryParam('1')).toBeTrue();
      expect(parseBooleanQueryParam('yes')).toBeTrue();
      expect(parseBooleanQueryParam('YES')).toBeTrue();
      expect(parseBooleanQueryParam(' true ')).toBeTrue();
    });

    it('parses falsy boolean strings correctly', () => {
      expect(parseBooleanQueryParam('false')).toBeFalse();
      expect(parseBooleanQueryParam('FALSE')).toBeFalse();
      expect(parseBooleanQueryParam('0')).toBeFalse();
      expect(parseBooleanQueryParam('no')).toBeFalse();
      expect(parseBooleanQueryParam('NO')).toBeFalse();
      expect(parseBooleanQueryParam(' false ')).toBeFalse();
    });

    it('returns undefined for invalid or empty inputs', () => {
      expect(parseBooleanQueryParam(null)).toBeUndefined();
      expect(parseBooleanQueryParam(undefined)).toBeUndefined();
      expect(parseBooleanQueryParam('')).toBeUndefined();
      expect(parseBooleanQueryParam('unknown')).toBeUndefined();
      expect(parseBooleanQueryParam('2')).toBeUndefined();
      expect(parseBooleanQueryParam('-1')).toBeUndefined();
    });
  });

  describe('resolveUseBinaryFormat', () => {
    let originalEnvUseBinary: boolean | undefined;

    beforeEach(() => {
      originalEnvUseBinary = environment.useBinaryFormat;
    });

    afterEach(() => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat =
        originalEnvUseBinary;
    });

    it('defaults to false when unconfigured and query params are empty', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat =
        undefined;
      const result = resolveUseBinaryFormat({ searchString: '' });
      expect(result).toBeFalse();
    });

    it('uses environment.useBinaryFormat when no query parameter is provided', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = true;
      expect(resolveUseBinaryFormat({ searchString: '' })).toBeTrue();

      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = false;
      expect(resolveUseBinaryFormat({ searchString: '' })).toBeFalse();
    });

    it('uses explicit environmentUseBinaryFormat option over global environment', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = false;
      const result = resolveUseBinaryFormat({
        searchString: '',
        environmentUseBinaryFormat: true,
      });
      expect(result).toBeTrue();
    });

    it('overrides environment setting with URL query parameter', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = false;
      const resultTrue = resolveUseBinaryFormat({
        searchString: '?useBinaryFormat=true',
      });
      expect(resultTrue).toBeTrue();

      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = true;
      const resultFalse = resolveUseBinaryFormat({
        searchString: '?useBinaryFormat=false',
      });
      expect(resultFalse).toBeFalse();
    });

    it('supports 1 and 0 in URL query parameter', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = false;
      expect(
        resolveUseBinaryFormat({ searchString: '?useBinaryFormat=1' }),
      ).toBeTrue();

      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = true;
      expect(
        resolveUseBinaryFormat({ searchString: '?useBinaryFormat=0' }),
      ).toBeFalse();
    });

    it('falls back to environment when URL query parameter is unrecognized', () => {
      (environment as { useBinaryFormat?: boolean }).useBinaryFormat = true;
      const result = resolveUseBinaryFormat({
        searchString: '?useBinaryFormat=invalid',
      });
      expect(result).toBeTrue();
    });
  });
});
