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

import { TestBed } from '@angular/core/testing';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';
import {
  CelTimelineFilter,
  CelLogFilter,
} from 'src/app/store/domain/filter/cel-filter';
import { ExcludeNoLogsFilter } from 'src/app/store/domain/filter/other-filter';

describe('InspectionDataStoreV2', () => {
  let service: InspectionDataStoreV2;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [InspectionDataStoreV2, CelTimelineFilter, CelLogFilter, ExcludeNoLogsFilter],
    });
  });

  it('should load mock data asynchronously on creation', async () => {
    service = TestBed.inject(InspectionDataStoreV2);
    service.loadMockData();

    const waitForSignal = async (): Promise<void> => {
      for (let i = 0; i < 50; i++) {
        if (service.inspectionData() !== null) {
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
      throw new Error(
        'Timeout waiting for inspectionData signal to be populated',
      );
    };

    await waitForSignal();

    expect(service.inspectionData()).not.toBeNull();
    expect(service.timelineView()).not.toBeNull();
  });

  it('should update signal values when setNewInspectionData is called', async () => {
    service = TestBed.inject(InspectionDataStoreV2);
    service.loadMockData();

    const waitForSignal = async (): Promise<void> => {
      for (let i = 0; i < 50; i++) {
        if (service.inspectionData() !== null) {
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
      throw new Error(
        'Timeout waiting for inspectionData signal to be populated',
      );
    };

    await waitForSignal();

    const newMockData = await createMockInspectionDataV2();
    service.setNewInspectionData(newMockData);

    expect(service.inspectionData()).toBe(newMockData);
  });
});
