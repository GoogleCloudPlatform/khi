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
  DEFAULT_DOWNLOAD_CHUNK_SIZE_BYTES,
  DEFAULT_DOWNLOAD_MAX_CONCURRENCY,
  resolveDownloadConfig,
} from 'src/app/services/api/download-config-resolver';
import {
  DownloadEnvironmentConfig,
  environment,
} from 'src/environments/environment';

describe('download-config-resolver', () => {
  let originalEnvDownload: DownloadEnvironmentConfig;

  beforeEach(() => {
    originalEnvDownload = environment.download;
  });

  afterEach(() => {
    (environment as { download: DownloadEnvironmentConfig }).download =
      originalEnvDownload;
  });

  it('uses default values when unconfigured and query params are empty', () => {
    (environment as { download: DownloadEnvironmentConfig }).download = {};
    const config = resolveDownloadConfig({ searchString: '' });
    expect(config.chunkSize).toBe(DEFAULT_DOWNLOAD_CHUNK_SIZE_BYTES);
    expect(config.maxConcurrency).toBe(DEFAULT_DOWNLOAD_MAX_CONCURRENCY);
  });

  it('uses environment configuration when present', () => {
    (environment as { download: DownloadEnvironmentConfig }).download = {
      chunkSizeBytes: 2 * 1024 * 1024,
      maxConcurrency: 6,
    };
    const config = resolveDownloadConfig({ searchString: '' });
    expect(config.chunkSize).toBe(2 * 1024 * 1024);
    expect(config.maxConcurrency).toBe(6);
  });

  it('overrides environment settings with URL query parameters', () => {
    (environment as { download: DownloadEnvironmentConfig }).download = {
      chunkSizeBytes: 8 * 1024 * 1024,
      maxConcurrency: 8,
    };
    const config = resolveDownloadConfig({
      searchString: '?downloadChunkSize=1MB&downloadConcurrency=3',
    });
    expect(config.chunkSize).toBe(1024 * 1024);
    expect(config.maxConcurrency).toBe(3);
  });

  it('parses raw byte count in URL query parameter', () => {
    const config = resolveDownloadConfig({
      searchString: '?downloadChunkSize=5242880',
    });
    expect(config.chunkSize).toBe(5242880);
  });

  it('falls back to environment or default when query parameter is invalid', () => {
    (environment as { download: DownloadEnvironmentConfig }).download = {
      chunkSizeBytes: 4 * 1024 * 1024,
      maxConcurrency: 5,
    };
    const config = resolveDownloadConfig({
      searchString: '?downloadChunkSize=invalid&downloadConcurrency=abc',
    });
    expect(config.chunkSize).toBe(4 * 1024 * 1024);
    expect(config.maxConcurrency).toBe(5);
  });

  it('clamps values to a minimum of 1', () => {
    const config = resolveDownloadConfig({
      searchString: '?downloadChunkSize=0&downloadConcurrency=0',
      environmentConfig: {
        chunkSizeBytes: 0,
        maxConcurrency: 0,
      },
    });
    // If parsed as 0, it falls back to defaults (which are > 1)
    expect(config.chunkSize).toBe(DEFAULT_DOWNLOAD_CHUNK_SIZE_BYTES);
    expect(config.maxConcurrency).toBe(DEFAULT_DOWNLOAD_MAX_CONCURRENCY);
  });
});
