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
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { ImportInspectionClientService } from 'src/app/services/api/import-inspection-client.service';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendSyncService } from 'src/app/services/api/backend-sync-interface';

describe('InspectionDataLoaderService', () => {
  let service: InspectionDataLoaderService;
  let progressSpy: jasmine.SpyObj<ProgressDialogStatusUpdator>;
  let importClientSpy: jasmine.SpyObj<ImportInspectionClientService>;
  let backendSyncSpy: jasmine.SpyObj<BackendSyncService>;

  beforeEach(() => {
    progressSpy = jasmine.createSpyObj<ProgressDialogStatusUpdator>(
      'ProgressDialogStatusUpdator',
      ['show', 'updateProgress', 'dismiss'],
    );

    importClientSpy = jasmine.createSpyObj<ImportInspectionClientService>(
      'ImportInspectionClientService',
      ['importFile'],
    );

    backendSyncSpy = {
      tasks: jasmine.createSpyObj('tasks', ['reload']),
    } as unknown as jasmine.SpyObj<BackendSyncService>;

    TestBed.configureTestingModule({
      providers: [
        InspectionDataLoaderService,
        InspectionDataStore,
        { provide: PROGRESS_DIALOG_STATUS_UPDATOR, useValue: progressSpy },
        { provide: BACKEND_API, useValue: {} as BackendAPI },
        { provide: EXTENSION_STORE, useValue: new ExtensionStore() },
        { provide: ImportInspectionClientService, useValue: importClientSpy },
        { provide: BACKEND_SYNC, useValue: backendSyncSpy },
      ],
    });

    service = TestBed.inject(InspectionDataLoaderService);
  });

  it('imports file via importClient and reloads tasks without opening inspection directly', async () => {
    importClientSpy.importFile.and.returnValue(
      Promise.resolve({
        inspectionId: 'test-inspection-id',
        inspectionName: 'Imported Cluster',
        fileSizeBytes: 100,
      }),
    );

    const testFile = new File([new Uint8Array(100)], 'test.khi');
    await service.importInspectionFile(testFile);

    expect(progressSpy.show).toHaveBeenCalled();
    expect(importClientSpy.importFile).toHaveBeenCalledWith(
      testFile,
      jasmine.any(Object),
    );
    expect(backendSyncSpy.tasks.reload).toHaveBeenCalled();
    expect(progressSpy.dismiss).toHaveBeenCalled();
  });

  it('dismisses progress dialog and alerts if import fails', async () => {
    importClientSpy.importFile.and.returnValue(
      Promise.reject(new Error('Validation error')),
    );
    spyOn(window, 'alert');

    const testFile = new File([new Uint8Array(100)], 'broken.khi');
    await service.importInspectionFile(testFile);

    expect(progressSpy.dismiss).toHaveBeenCalled();
    expect(window.alert).toHaveBeenCalledWith(
      jasmine.stringMatching(
        /Failed to import inspection file: Validation error/,
      ),
    );
  });
});
