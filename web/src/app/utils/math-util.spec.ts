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

import { clampNumber } from 'src/app/utils/math-util';

describe('math-util', () => {
  describe('clampNumber', () => {
    interface ClampTestCase {
      readonly desc: string;
      readonly value: number;
      readonly min: number;
      readonly max: number;
      readonly want: number;
    }

    const testCases: readonly ClampTestCase[] = [
      {
        desc: 'returns value when it is strictly within bounds',
        value: 15,
        min: 10,
        max: 20,
        want: 15,
      },
      {
        desc: 'clamps to min when value is below lower bound',
        value: 5,
        min: 10,
        max: 20,
        want: 10,
      },
      {
        desc: 'clamps to max when value is above upper bound',
        value: 25,
        min: 10,
        max: 20,
        want: 20,
      },
      {
        desc: 'returns min when value is exactly equal to min',
        value: 10,
        min: 10,
        max: 20,
        want: 10,
      },
      {
        desc: 'returns max when value is exactly equal to max',
        value: 20,
        min: 10,
        max: 20,
        want: 20,
      },
      {
        desc: 'returns min when min equals max and value differs',
        value: 50,
        min: 10,
        max: 10,
        want: 10,
      },
      {
        desc: 'returns min when max is smaller than min',
        value: 15,
        min: 20,
        max: 10,
        want: 20,
      },
    ];

    for (const tc of testCases) {
      it(`should ${tc.desc}`, () => {
        expect(clampNumber(tc.value, tc.min, tc.max)).toBe(tc.want);
      });
    }
  });
});
