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
  diffStrings,
  DiffStatus,
} from 'src/app/shared/components/yaml-viewer/lcs';

describe('diffStrings', () => {
  it('should return unchanged segments for identical strings', () => {
    const result = diffStrings('hello', 'hello');
    expect(result.oldSegs).toEqual([
      { text: 'hello', diffStatus: DiffStatus.Unchanged },
    ]);
    expect(result.newSegs).toEqual([
      { text: 'hello', diffStatus: DiffStatus.Unchanged },
    ]);
  });

  it('should detect additions', () => {
    const result = diffStrings('hello', 'hello world');
    expect(result.oldSegs).toEqual([
      { text: 'hello', diffStatus: DiffStatus.Unchanged },
    ]);
    expect(result.newSegs).toEqual([
      { text: 'hell', diffStatus: DiffStatus.Unchanged },
      { text: 'o w', diffStatus: DiffStatus.Added },
      { text: 'o', diffStatus: DiffStatus.Unchanged },
      { text: 'rld', diffStatus: DiffStatus.Added },
    ]);
  });

  it('should detect deletions', () => {
    const result = diffStrings('hello world', 'hello');
    expect(result.oldSegs).toEqual([
      { text: 'hell', diffStatus: DiffStatus.Unchanged },
      { text: 'o w', diffStatus: DiffStatus.Deleted },
      { text: 'o', diffStatus: DiffStatus.Unchanged },
      { text: 'rld', diffStatus: DiffStatus.Deleted },
    ]);
    expect(result.newSegs).toEqual([
      { text: 'hello', diffStatus: DiffStatus.Unchanged },
    ]);
  });

  it('should detect modifications as deletion and addition', () => {
    const result = diffStrings('hello world', 'hello earth');
    expect(result.oldSegs).toEqual([
      { text: 'hello ', diffStatus: DiffStatus.Unchanged },
      { text: 'wo', diffStatus: DiffStatus.Deleted },
      { text: 'r', diffStatus: DiffStatus.Unchanged },
      { text: 'ld', diffStatus: DiffStatus.Deleted },
    ]);
    expect(result.newSegs).toEqual([
      { text: 'hello ', diffStatus: DiffStatus.Unchanged },
      { text: 'ea', diffStatus: DiffStatus.Added },
      { text: 'r', diffStatus: DiffStatus.Unchanged },
      { text: 'th', diffStatus: DiffStatus.Added },
    ]);
  });

  it('should handle empty strings', () => {
    const result = diffStrings('', 'hello');
    expect(result.oldSegs).toEqual([]);
    expect(result.newSegs).toEqual([
      { text: 'hello', diffStatus: DiffStatus.Added },
    ]);
  });
});
