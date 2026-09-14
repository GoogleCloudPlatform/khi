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

import { formatBytes } from 'src/app/utils/byte-format-util';

describe('byte-format-util', () => {
  describe('formatBytes', () => {
    it('should format 0 and negative bytes as 0 B', () => {
      expect(formatBytes(0)).toBe('0 B');
      expect(formatBytes(-100)).toBe('0 B');
    });

    it('should format non-finite numbers as 0 B', () => {
      expect(formatBytes(NaN)).toBe('0 B');
      expect(formatBytes(Infinity)).toBe('0 B');
      expect(formatBytes(-Infinity)).toBe('0 B');
    });

    it('should format small bytes as B', () => {
      expect(formatBytes(512)).toBe('512 B');
    });

    it('should format kilobytes, megabytes, gigabytes, and terabytes', () => {
      expect(formatBytes(1024)).toBe('1.0 KB');
      expect(formatBytes(1536)).toBe('1.5 KB');
      expect(formatBytes(1048576)).toBe('1.0 MB');
      expect(formatBytes(10485760)).toBe('10 MB');
      expect(formatBytes(1073741824)).toBe('1.0 GB');
      expect(formatBytes(1099511627776)).toBe('1.0 TB');
    });
  });
});
