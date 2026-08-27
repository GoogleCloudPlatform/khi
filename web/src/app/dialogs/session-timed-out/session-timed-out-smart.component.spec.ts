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
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { By } from '@angular/platform-browser';
import { signal } from '@angular/core';
import {
  openSessionTimedOutDialog,
  SessionTimedOutSmartComponent,
} from 'src/app/dialogs/session-timed-out/session-timed-out-smart.component';
import { SessionTimedOutLayoutComponent } from 'src/app/dialogs/session-timed-out/components/session-timed-out-layout.component';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';

describe('SessionTimedOutSmartComponent', () => {
  let component: SessionTimedOutSmartComponent;
  let fixture: ComponentFixture<SessionTimedOutSmartComponent>;
  let dialogRefSpy: jasmine.SpyObj<
    MatDialogRef<SessionTimedOutSmartComponent, void>
  >;
  let dialogSpy: jasmine.SpyObj<MatDialog>;
  let workbenchClientSpy: jasmine.SpyObj<WorkbenchClientService>;
  let progressDialogSpy: jasmine.SpyObj<ProgressDialogStatusUpdator>;

  beforeEach(async () => {
    dialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close']);
    dialogSpy = jasmine.createSpyObj('MatDialog', ['open']);
    workbenchClientSpy = jasmine.createSpyObj(
      'WorkbenchClientService',
      ['reopenWorkbench', 'closeWorkbench'],
      {
        isReopening: signal(false).asReadonly(),
      },
    );
    progressDialogSpy = jasmine.createSpyObj('ProgressDialogStatusUpdator', [
      'show',
      'dismiss',
      'updateProgress',
    ]);

    TestBed.overrideProvider(MatDialog, { useValue: dialogSpy });

    await TestBed.configureTestingModule({
      imports: [SessionTimedOutSmartComponent, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: WorkbenchClientService, useValue: workbenchClientSpy },
        {
          provide: PROGRESS_DIALOG_STATUS_UPDATOR,
          useValue: progressDialogSpy,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(SessionTimedOutSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and render layout component', () => {
    expect(component).toBeTruthy();
    const layoutEl = fixture.debugElement.query(
      By.directive(SessionTimedOutLayoutComponent),
    );
    expect(layoutEl).toBeTruthy();
  });

  it('should show progress, reconnect session, and close dialog on success', async () => {
    workbenchClientSpy.reopenWorkbench.and.callFake((onProgress) => {
      onProgress?.('Reading...', 50, 2);
      return Promise.resolve('wb-restored');
    });

    const layoutEl = fixture.debugElement.query(
      By.directive(SessionTimedOutLayoutComponent),
    );
    layoutEl.triggerEventHandler('reconnect', undefined);
    await fixture.whenStable();

    expect(progressDialogSpy.show).toHaveBeenCalledOnceWith();
    expect(progressDialogSpy.updateProgress).toHaveBeenCalled();
    expect(progressDialogSpy.dismiss).toHaveBeenCalledOnceWith();
    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith();
  });

  it('should dismiss progress and update error message on reconnection failure', async () => {
    workbenchClientSpy.reopenWorkbench.and.rejectWith(
      new Error('Connection failed'),
    );

    const layoutEl = fixture.debugElement.query(
      By.directive(SessionTimedOutLayoutComponent),
    );
    layoutEl.triggerEventHandler('reconnect', undefined);
    await fixture.whenStable();
    fixture.detectChanges();

    expect(progressDialogSpy.show).toHaveBeenCalledOnceWith();
    expect(progressDialogSpy.dismiss).toHaveBeenCalledOnceWith();
    expect(dialogRefSpy.close).not.toHaveBeenCalled();

    const layoutInstance =
      layoutEl.componentInstance as SessionTimedOutLayoutComponent;
    expect(layoutInstance.errorMessage()).toContain('Connection failed');
  });

  it('should close workbench, close dialog, and open startup dialog when returnToStartup is triggered', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(SessionTimedOutLayoutComponent),
    );
    layoutEl.triggerEventHandler('returnToStartup', undefined);

    expect(workbenchClientSpy.closeWorkbench).toHaveBeenCalledOnceWith();
    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith();
    expect(dialogSpy.open).toHaveBeenCalled();
  });

  describe('openSessionTimedOutDialog', () => {
    it('should open dialog with disableClose set to true', () => {
      openSessionTimedOutDialog(dialogSpy);
      expect(dialogSpy.open).toHaveBeenCalledWith(
        SessionTimedOutSmartComponent,
        jasmine.objectContaining({
          disableClose: true,
        }),
      );
    });
  });
});
