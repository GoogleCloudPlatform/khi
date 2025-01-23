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

import { ProgressUtil } from './progress-util';

describe('ProgressUtil', () => {
  describe('formatPogressMessageByBytes', () => {
    it('should format progress message correctly', () => {
      expect(ProgressUtil.formatPogressMessageByBytes(1024, 2048)).toBe(
        '1,024 Bytes of 2,048 Bytes(50.0%)',
      );
      expect(ProgressUtil.formatPogressMessageByBytes(0, 2048)).toBe(
        '0 Bytes of 2,048 Bytes(0.0%)',
      );
      expect(ProgressUtil.formatPogressMessageByBytes(2048, 2048)).toBe(
        '2,048 Bytes of 2,048 Bytes(100.0%)',
      );
      expect(ProgressUtil.formatPogressMessageByBytes(1234567, 2469134)).toBe(
        '1,234,567 Bytes of 2,469,134 Bytes(50.0%)',
      );

      // Handle cases where 'all' is zero to prevent division by zero
      expect(ProgressUtil.formatPogressMessageByBytes(1024, 0)).toBe(
        '1,024 Bytes of 0 Bytes(Infinity%)',
      );
    });
  });
});
