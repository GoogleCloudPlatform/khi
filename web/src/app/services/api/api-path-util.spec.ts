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

import { ApiPathUtil } from 'src/app/services/api/api-path-util';

describe('ApiPathUtil', () => {
  let metaTag: HTMLMetaElement | null = null;

  afterEach(() => {
    if (metaTag && metaTag.parentNode) {
      metaTag.parentNode.removeChild(metaTag);
      metaTag = null;
    }
  });

  function setMetaTag(content: string | null) {
    metaTag = document.getElementById(
      'server-base-path',
    ) as HTMLMetaElement | null;
    if (!metaTag) {
      metaTag = document.createElement('meta');
      metaTag.id = 'server-base-path';
      document.head.appendChild(metaTag);
    }
    if (content !== null) {
      metaTag.setAttribute('content', content);
    } else {
      metaTag.removeAttribute('content');
    }
  }

  describe('getServerBasePath', () => {
    it('returns empty string when meta tag does not exist', () => {
      const existingTag = document.getElementById('server-base-path');
      if (existingTag && existingTag.parentNode) {
        existingTag.parentNode.removeChild(existingTag);
      }
      expect(ApiPathUtil.getServerBasePath()).toBe('');
    });

    it('returns trimmed base path without trailing slash', () => {
      setMetaTag('/custom/prefix/');
      expect(ApiPathUtil.getServerBasePath()).toBe('/custom/prefix');
    });

    it('returns base path unchanged when no trailing slash', () => {
      setMetaTag('/custom/prefix');
      expect(ApiPathUtil.getServerBasePath()).toBe('/custom/prefix');
    });
  });

  describe('getServerBaseUrl', () => {
    it('returns window.location.origin when meta tag is not set', () => {
      const existingTag = document.getElementById('server-base-path');
      if (existingTag && existingTag.parentNode) {
        existingTag.parentNode.removeChild(existingTag);
      }
      expect(ApiPathUtil.getServerBaseUrl()).toBe(window.location.origin);
    });

    it('returns absolute URL directly when meta tag contains http scheme', () => {
      setMetaTag('http://localhost:8080');
      expect(ApiPathUtil.getServerBaseUrl()).toBe('http://localhost:8080');
    });

    it('returns absolute URL directly when meta tag contains https scheme', () => {
      setMetaTag('https://example.com/api');
      expect(ApiPathUtil.getServerBaseUrl()).toBe('https://example.com/api');
    });

    it('prepends origin when meta tag is a relative subpath', () => {
      setMetaTag('/subpath');
      expect(ApiPathUtil.getServerBaseUrl()).toBe(
        `${window.location.origin}/subpath`,
      );
    });
  });
});
