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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StartupDialogSmartComponent } from './startup-smart.component';
import { StartupDialogLayoutComponent } from 'src/app/dialogs/startup/components/startup-dialog-layout.component';
import { signal } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { InspectionClient } from 'src/app/services/api/backend-api.service';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { ProgressDialogService } from 'src/app/services/progress/progress-dialog.service';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendConnectionStatus } from 'src/app/services/api/backend-sync-interface';
import { InspectionListItem } from 'src/app/generated/api/v1/inspection_pb';
import {
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';

import { of } from 'rxjs';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import { By } from '@angular/platform-browser';

describe('StartupDialogComponent', () => {
  let component: ComponentFixture<StartupDialogSmartComponent>;

  let backendAPISpy: jasmine.SpyObj<BackendAPI>;
  let dialogSpy: jasmine.SpyObj<MatDialog>;
  let mockInspectionClient: {
    inspectionID: string;
    dryrunDirect: jasmine.Spy;
    run: jasmine.Spy;
  };

  beforeEach(async () => {
    backendAPISpy = jasmine.createSpyObj<BackendAPI>('BackendAPIService', [
      'patchInspection',
      'createInspection',
      'setEnabledFeatures',
    ]);
    backendAPISpy.patchInspection.and.returnValue(of());
    mockInspectionClient = {
      inspectionID: 'test-inspection-id',
      dryrunDirect: jasmine.createSpy('dryrunDirect').and.returnValue(
        of({
          metadata: {
            form: [],
            query: [],
            plan: { tasks: [] },
          },
        }),
      ),
      run: jasmine.createSpy('run').and.returnValue(of(undefined)),
    };
    backendAPISpy.createInspection.and.returnValue(
      of(mockInspectionClient as unknown as InspectionClient),
    );
    backendAPISpy.setEnabledFeatures.and.returnValue(of(undefined));

    dialogSpy = jasmine.createSpyObj<MatDialog>('MatDialog', ['open']);

    TestBed.configureTestingModule({
      providers: [
        ...ProgressDialogService.providers(),
        {
          provide: MatDialogRef,
          useValue: {},
        },
        {
          provide: MatDialog,
          useValue: dialogSpy,
        },
        {
          provide: BACKEND_API,
          useValue: backendAPISpy,
        },
        {
          provide: BACKEND_SYNC,
          useValue: {
            inspections: signal<readonly InspectionListItem[]>([]),
            connectionStatus: signal(BackendConnectionStatus.Connected),
          },
        },
        {
          provide: EXTENSION_STORE,
          useValue: new ExtensionStore(),
        },
        {
          provide: InspectionDataLoaderService,
          useClass: InspectionDataLoaderService,
        },
      ],
    });
    component = TestBed.createComponent(StartupDialogSmartComponent);
    component.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should save title to backend', () => {
    const layoutEl = component.debugElement.query(
      By.directive(StartupDialogLayoutComponent),
    );
    layoutEl.triggerEventHandler('changeInspectionTitle', {
      id: 'test-task',
      changeTo: 'New Title',
    });
    expect(backendAPISpy.patchInspection).toHaveBeenCalledOnceWith(
      'test-task',
      {
        name: 'New Title',
      },
    );
  });

  it('should do nothing when Job Command dialog is dismissed without input', async () => {
    const dialogRefSpy = {
      afterClosed: jasmine
        .createSpy('afterClosed')
        .and.returnValue(of(undefined)),
    };
    dialogSpy.open.and.returnValue(
      dialogRefSpy as unknown as MatDialogRef<unknown>,
    );

    const layoutEl = component.debugElement.query(
      By.directive(StartupDialogLayoutComponent),
    );
    layoutEl.triggerEventHandler('startFromJobCommand', null);
    await component.whenStable();

    expect(dialogSpy.open).toHaveBeenCalled();
    expect(backendAPISpy.createInspection).not.toHaveBeenCalled();
  });

  it('should create, configure features, dryrun, and run inspection when dryrun has no errors', async () => {
    const parsedCommand = {
      inspectionType: 'gke',
      features: ['feature-a'],
      parameters: { cluster: 'my-cluster' },
    };
    const dialogRefSpy = {
      afterClosed: jasmine
        .createSpy('afterClosed')
        .and.returnValue(of(parsedCommand)),
    };
    dialogSpy.open.and.returnValue(
      dialogRefSpy as unknown as MatDialogRef<unknown>,
    );

    const layoutEl = component.debugElement.query(
      By.directive(StartupDialogLayoutComponent),
    );
    layoutEl.triggerEventHandler('startFromJobCommand', null);
    await component.whenStable();

    expect(backendAPISpy.createInspection).toHaveBeenCalledWith('gke');
    expect(backendAPISpy.setEnabledFeatures).toHaveBeenCalledWith(
      'test-inspection-id',
      { 'feature-a': true },
    );
    expect(mockInspectionClient.dryrunDirect).toHaveBeenCalledWith({
      cluster: 'my-cluster',
    });
    expect(mockInspectionClient.run).toHaveBeenCalledWith({
      cluster: 'my-cluster',
    });
  });

  it('should fall back to opening NewInspectionDialogComponent when dryrun has errors', async () => {
    const parsedCommand = {
      inspectionType: 'gke',
      features: ['feature-a'],
      parameters: { cluster: 'my-cluster' },
    };
    const dialogRefSpy = {
      afterClosed: jasmine
        .createSpy('afterClosed')
        .and.returnValue(of(parsedCommand)),
    };
    dialogSpy.open.and.returnValue(
      dialogRefSpy as unknown as MatDialogRef<unknown>,
    );

    mockInspectionClient.dryrunDirect.and.returnValue(
      of({
        metadata: {
          form: [
            {
              id: 'cluster',
              type: ParameterInputType.Text,
              label: 'Cluster',
              description: '',
              hint: 'Required',
              hintType: ParameterHintType.Error,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            },
          ],
          query: [],
          plan: { tasks: [] },
        },
      }),
    );

    const layoutEl = component.debugElement.query(
      By.directive(StartupDialogLayoutComponent),
    );
    layoutEl.triggerEventHandler('startFromJobCommand', null);
    await component.whenStable();

    expect(backendAPISpy.createInspection).toHaveBeenCalledWith('gke');
    expect(mockInspectionClient.run).not.toHaveBeenCalled();
    expect(dialogSpy.open).toHaveBeenCalledTimes(2);
  });
});
