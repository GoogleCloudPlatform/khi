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

import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';

describe('inspection-data.mock', () => {
  describe('createMockInspectionDataV2', () => {
    it('should create a valid mock instance of InspectionDataV2 with timeline hierarchy and loaded icons', async () => {
      const mockData = await createMockInspectionDataV2();

      expect(mockData).toBeDefined();
      expect(mockData.internPool).toBeDefined();
      expect(mockData.styleStore).toBeDefined();
      expect(mockData.logStore).toBeDefined();
      expect(mockData.timelineStore).toBeDefined();
      expect(mockData.metadata).toBeDefined();

      const timelines = mockData.timelineStore.timelines;
      expect(timelines.length).toBe(111121);

      const leaf = mockData.timelineStore.getTimeline(4);
      expect(leaf.id).toBe(4);
      expect(leaf.name).toBe('mock-pod-1');
      expect(leaf.parent?.id).toBe(3);

      expect(mockData.logStore.count).toBe(130001);

      const log = mockData.logStore.getLog(1);
      expect(log.body).toEqual({
        message: 'Pod created successfully',
        reason: 'Created',
        source: { component: 'kubelet', host: 'node-1' },
      });
    });
  });
});
