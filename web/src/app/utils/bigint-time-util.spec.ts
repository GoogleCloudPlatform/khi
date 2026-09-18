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

  describe('NsToProtoTimestamp', () => {
    it('should convert nanoseconds to protobuf Timestamp message', () => {
      const ts0 = BigIntTimeUtil.NsToProtoTimestamp(0n);
      expect(ts0.seconds).toBe(0n);
      expect(ts0.nanos).toBe(0);

      const ts1 = BigIntTimeUtil.NsToProtoTimestamp(1_000_000_000n);
      expect(ts1.seconds).toBe(1n);
      expect(ts1.nanos).toBe(0);

      const ts2 = BigIntTimeUtil.NsToProtoTimestamp(1_234_567_890n);
      expect(ts2.seconds).toBe(1n);
      expect(ts2.nanos).toBe(234567890);

      const ts3 = BigIntTimeUtil.NsToProtoTimestamp(1700000000123456789n);
      expect(ts3.seconds).toBe(1700000000n);
      expect(ts3.nanos).toBe(123456789);

      const tsNeg = BigIntTimeUtil.NsToProtoTimestamp(-1_500_000_000n);
      expect(tsNeg.seconds).toBe(-2n);
      expect(tsNeg.nanos).toBe(500000000);
    });
  });
});
