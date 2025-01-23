/**
 * Copyright 2024 Google LLC
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
  filteElementsByIncludedSubstring,
  subtractSet,
} from './collection-util';

describe('Collection util', () => {
  it('subtractSet', () => {
    const subtractFrom = new Set(['a', 'b', 'c', 'd']);
    const subtracting = new Set(['a', 'c', 'e']);
    const result = subtractSet(subtractFrom, subtracting);
    expect(result.has('a')).toBeFalse();
    expect(result.has('b')).toBeTrue();
    expect(result.has('c')).toBeFalse();
    expect(result.has('d')).toBeTrue();
    expect(result.has('e')).toBeFalse();

    const subtractFrom2 = new Set(['a']);
    const subtracting2 = new Set([]);
    const result2 = subtractSet(subtractFrom2, subtracting2);
    expect(result2.has('a')).toBeTrue();
  });

  it('filteElementsByIncludedSubstring', () => {
    const result = filteElementsByIncludedSubstring(
      [
        'keyword',
        'keywordB',
        'not-key-word',
        'Akeyword',
        'keywordA',
        'Bkeyword',
        'yet-another-not-key-word',
      ],
      'keyword',
    );
    expect(result.length).toBe(5);
    expect(result[0]).toBe('keyword');
    expect(result[1]).toBe('keywordA');
    expect(result[2]).toBe('keywordB');
    expect(result[3]).toBe('Akeyword');
    expect(result[4]).toBe('Bkeyword');
  });
});
