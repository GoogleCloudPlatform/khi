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

import { ComponentFixture, fakeAsync, TestBed } from '@angular/core/testing';
import { StartupDialogComponent } from './startup.component';
import { MatDialogRef } from '@angular/material/dialog';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { ProgressDialogService } from 'src/app/services/progress/progress-dialog.service';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import { ReplaySubject, Subject } from 'rxjs';
import { By } from '@angular/platform-browser';
import { GetInspectionTasksResponse } from 'src/app/common/schema/api-types';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

describe('StartupDialogComponent', () => {
  let component: ComponentFixture<StartupDialogComponent>;
  let backendConnectionSpy: jasmine.SpyObj<BackendConnectionService>;
  let taskListSubject: Subject<GetInspectionTasksResponse>;
  beforeEach(async () => {
    taskListSubject = new ReplaySubject(1);
    backendConnectionSpy = jasmine.createSpyObj<BackendConnectionService>(
      'BackendConnectionService',
      ['tasks'],
    );
    backendConnectionSpy.tasks.and.returnValue(taskListSubject);
    await TestBed.configureTestingModule({
      imports: [StartupDialogComponent],
      providers: [
        ...ProgressDialogService.providers(),
        {
          provide: MatDialogRef,
          useValue: {},
        },
        {
          provide: BACKEND_API,
          useValue: {},
        },
        {
          provide: BACKEND_CONNECTION,
          useValue: backendConnectionSpy,
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
    component = TestBed.createComponent(StartupDialogComponent);
    component.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should show loading message when task lists are not loaded yet', () => {
    expect(
      component.debugElement.query(By.css('.task-list-loading > p'))
        .nativeElement.innerText,
    ).toBe('Loading task list...');
  });

  it('should show empty list with hint message when given task list is empty', fakeAsync(() => {
    taskListSubject.next({
      tasks: {
        a: {
          header: {
            inspectionType: 'foo',
            inspectionTypeIconPath: '',
            inspectTimeUnixSeconds: 0,
            startTimeUnixSeconds: 0,
            endTimeUnixSeconds: 0,
            suggestedFilename: '',
          },
          progress: {
            totalProgress: {
              id: 'foo',
              label: 'total',
              percentage: 10,
              message: 'progress-foo',
              indeterminate: false,
            },
            progresses: [],
            phase: 'DONE',
          },
          error: {
            errorMessages: [],
          },
        },
      },
      serverStat: {
        totalMemoryAvailable: 0,
      },
    });
    component.detectChanges();
    expect(backendConnectionSpy.tasks).toHaveBeenCalled();

    const taskElements = component.debugElement.queryAll(
      By.css('.inspection-task'),
    );
    expect(taskElements.length).toBe(1);
  }));
});
