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

import { BigIntTimeUtil } from './bigint-time-util';

describe('BigIntTimeUtil', () => {
  describe('NsToNumberMs', () => {
    it('should convert nanoseconds to milliseconds', () => {
      expect(BigIntTimeUtil.NsToNumberMs(0n)).toBe(0);
      expect(BigIntTimeUtil.NsToNumberMs(1000000n)).toBe(1);
      expect(BigIntTimeUtil.NsToNumberMs(1500000n)).toBe(1.5);
      expect(BigIntTimeUtil.NsToNumberMs(1234567890n)).toBe(1234.56789);
      expect(BigIntTimeUtil.NsToNumberMs(500n)).toBe(0.0005);
    });
  });
});
