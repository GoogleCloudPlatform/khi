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

/**
 * Utility functions for resolving server base paths and URLs across API clients.
 */
export class ApiPathUtil {
  /**
   * Retrieves the raw server base path configured in the DOM meta tag.
   * @returns The trimmed base path string, or empty string if not found.
   */
  public static getServerBasePath(): string {
    const basePathTag = document.getElementById('server-base-path');
    if (basePathTag === null) {
      return '';
    }
    let content = basePathTag.getAttribute('content');
    if (content?.endsWith('/')) {
      content = content.substring(0, content.length - 1);
    }
    return content ?? '';
  }

  /**
   * Resolves the full base URL for API/RPC requests, accounting for dev servers and reverse proxy base paths.
   * @returns Fully qualified base URL.
   */
  public static getServerBaseUrl(): string {
    const serverBasePath = ApiPathUtil.getServerBasePath();
    if (!serverBasePath) {
      return window.location.origin;
    }
    if (
      serverBasePath.startsWith('http://') ||
      serverBasePath.startsWith('https://')
    ) {
      return serverBasePath;
    }
    return `${window.location.origin}${serverBasePath}`;
  }
}
