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
import { of, throwError } from 'rxjs';
import { InspectionDataLoaderService } from './data-loader.service';
import { BACKEND_API, BackendAPI } from './api/backend-api-interface';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from './progress/progress-interface';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from '../extensions/extension-common/extension-store';
import { InspectionDataStore } from './inspection-data-store.service';
import { ImportInspectionClientService } from './api/import-inspection-client.service';
import { WorkbenchClientService } from './api/workbench/workbench-client.service';
import { KHIFileParser } from 'src/app/parser/core/file-parser';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';
import { OpenWorkbenchResponse_Stage } from 'src/app/generated/api/v1/workbench_pb';

describe('InspectionDataLoaderService', () => {
  let service: InspectionDataLoaderService;
  let mockProgress: jasmine.SpyObj<ProgressDialogStatusUpdator>;
  let mockInspectionDataStore: jasmine.SpyObj<InspectionDataStore>;
  let mockBackendService: jasmine.SpyObj<BackendAPI>;
  let mockExtensionStore: jasmine.SpyObj<ExtensionStore>;
  let mockImportClient: jasmine.SpyObj<ImportInspectionClientService>;
  let mockWorkbenchClient: jasmine.SpyObj<WorkbenchClientService>;

  beforeEach(() => {
    mockProgress = jasmine.createSpyObj<ProgressDialogStatusUpdator>([
      'show',
      'dismiss',
      'updateProgress',
    ]);
    mockInspectionDataStore = jasmine.createSpyObj<InspectionDataStore>([
      'setNewInspectionData',
    ]);
    mockBackendService = jasmine.createSpyObj<BackendAPI>([
      'getInspectionData',
    ]);
    mockExtensionStore = jasmine.createSpyObj<ExtensionStore>([
      'notifyLifecycleOnInspectionDataOpen',
    ]);
    mockImportClient = jasmine.createSpyObj<ImportInspectionClientService>([
      'importFile',
    ]);
    mockWorkbenchClient = jasmine.createSpyObj<WorkbenchClientService>([
      'openWorkbench',
    ]);

    TestBed.configureTestingModule({
      providers: [
        InspectionDataLoaderService,
        {
          provide: PROGRESS_DIALOG_STATUS_UPDATOR,
          useValue: mockProgress,
        },
        {
          provide: InspectionDataStore,
          useValue: mockInspectionDataStore,
        },
        {
          provide: BACKEND_API,
          useValue: mockBackendService,
        },
        {
          provide: EXTENSION_STORE,
          useValue: mockExtensionStore,
        },
        {
          provide: ImportInspectionClientService,
          useValue: mockImportClient,
        },
        {
          provide: WorkbenchClientService,
          useValue: mockWorkbenchClient,
        },
      ],
    });

    service = TestBed.inject(InspectionDataLoaderService);
  });

  describe('importInspectionFile', () => {
    it('should show progress and call importClient.importFile', async () => {
      const file = new File(['test'], 'test.khi');
      mockImportClient.importFile.and.resolveTo();

      await service.importInspectionFile(file);

      expect(mockProgress.show).toHaveBeenCalled();
      expect(mockImportClient.importFile).toHaveBeenCalledWith(
        file,
        jasmine.any(Object),
      );
      expect(mockProgress.dismiss).toHaveBeenCalled();
    });

    it('should dismiss progress and alert on import error', async () => {
      const alertSpy = spyOn(window, 'alert').and.stub();
      spyOn(console, 'error').and.stub();
      const file = new File(['test'], 'test.khi');
      mockImportClient.importFile.and.rejectWith(new Error('Upload failed'));

      await service.importInspectionFile(file);

      expect(mockProgress.dismiss).toHaveBeenCalled();
      expect(alertSpy).toHaveBeenCalledWith(
        jasmine.stringMatching(
          /Failed to import inspection file: Upload failed/,
        ),
      );
    });
  });

  describe('loadInspectionDataFromBackend', () => {
    it('should wait for both local parsing and server openWorkbench before completing', async () => {
      const mockInspectionData = await createMockInspectionData();
      spyOn(KHIFileParser.prototype, 'parse').and.returnValue(
        Promise.resolve(mockInspectionData),
      );

      mockBackendService.getInspectionData.and.callFake(
        (_id, progressReporter) => {
          progressReporter(100, 100);
          return of({
            fileName: 'test.khi',
            content: new Blob(['fake-data']),
          });
        },
      );

      mockWorkbenchClient.openWorkbench.and.callFake(
        async (_session, _id, progressCallback) => {
          progressCallback?.(
            'Building search index...',
            100,
            OpenWorkbenchResponse_Stage.INDEXING_DATA,
          );
          return 'wb-123';
        },
      );

      await service.loadInspectionDataFromBackend('insp-1');

      expect(mockProgress.show).toHaveBeenCalled();
      expect(mockWorkbenchClient.openWorkbench).toHaveBeenCalled();
      expect(mockBackendService.getInspectionData).toHaveBeenCalled();
      expect(mockInspectionDataStore.setNewInspectionData).toHaveBeenCalledWith(
        mockInspectionData,
      );
      expect(
        mockExtensionStore.notifyLifecycleOnInspectionDataOpen,
      ).toHaveBeenCalled();
      expect(mockProgress.dismiss).toHaveBeenCalled();
    });

    it('should dismiss progress and alert on openWorkbench failure', async () => {
      const alertSpy = spyOn(window, 'alert').and.stub();
      spyOn(console, 'error').and.stub();

      mockBackendService.getInspectionData.and.returnValue(
        of({
          fileName: 'test.khi',
          content: new Blob(['fake-data']),
        }),
      );

      mockWorkbenchClient.openWorkbench.and.rejectWith(
        new Error('Server error'),
      );

      await service.loadInspectionDataFromBackend('insp-1');

      expect(mockProgress.dismiss).toHaveBeenCalled();
      expect(alertSpy).toHaveBeenCalled();
      expect(
        mockInspectionDataStore.setNewInspectionData,
      ).not.toHaveBeenCalled();
    });

    it('should dismiss progress and alert on getInspectionData failure', async () => {
      const alertSpy = spyOn(window, 'alert').and.stub();
      spyOn(console, 'error').and.stub();

      mockBackendService.getInspectionData.and.returnValue(
        throwError(() => new Error('Network error')),
      );

      mockWorkbenchClient.openWorkbench.and.resolveTo('wb-123');

      await service.loadInspectionDataFromBackend('insp-1');

      expect(mockProgress.dismiss).toHaveBeenCalled();
      expect(alertSpy).toHaveBeenCalled();
      expect(
        mockInspectionDataStore.setNewInspectionData,
      ).not.toHaveBeenCalled();
    });
  });
});
