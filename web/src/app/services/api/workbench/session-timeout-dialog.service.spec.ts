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
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { EMPTY } from 'rxjs';
import { signal } from '@angular/core';
import { SessionTimeoutDialogService } from 'src/app/services/api/workbench/session-timeout-dialog.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { SessionTimedOutSmartComponent } from 'src/app/dialogs/session-timed-out/session-timed-out-smart.component';

describe('SessionTimeoutDialogService', () => {
  let service: SessionTimeoutDialogService;
  let dialogSpy: jasmine.SpyObj<MatDialog>;
  let isWorkbenchExpiredSignal: ReturnType<typeof signal<boolean>>;
  let mockDialogRef: jasmine.SpyObj<
    MatDialogRef<SessionTimedOutSmartComponent, void>
  >;

  beforeEach(() => {
    isWorkbenchExpiredSignal = signal(false);
    mockDialogRef = jasmine.createSpyObj('MatDialogRef', [
      'close',
      'afterClosed',
    ]);
    mockDialogRef.afterClosed.and.returnValue(EMPTY);

    dialogSpy = jasmine.createSpyObj('MatDialog', ['open']);
    dialogSpy.open.and.returnValue(mockDialogRef);

    const mockWorkbenchClient = {
      isWorkbenchExpired: isWorkbenchExpiredSignal.asReadonly(),
    };

    TestBed.overrideProvider(MatDialog, { useValue: dialogSpy });

    TestBed.configureTestingModule({
      providers: [
        SessionTimeoutDialogService,
        { provide: WorkbenchClientService, useValue: mockWorkbenchClient },
      ],
    });

    service = TestBed.inject(SessionTimeoutDialogService);
  });

  it('should be created and not open dialog initially when session is active', () => {
    expect(service).toBeTruthy();
    expect(dialogSpy.open).not.toHaveBeenCalled();
  });

  it('should open dialog when isWorkbenchExpired transitions to true', () => {
    isWorkbenchExpiredSignal.set(true);
    TestBed.flushEffects();

    expect(dialogSpy.open).toHaveBeenCalledOnceWith(
      SessionTimedOutSmartComponent,
      jasmine.objectContaining({ disableClose: true }),
    );
  });

  it('should close dialog when isWorkbenchExpired becomes false', () => {
    isWorkbenchExpiredSignal.set(true);
    TestBed.flushEffects();

    isWorkbenchExpiredSignal.set(false);
    TestBed.flushEffects();

    expect(mockDialogRef.close).toHaveBeenCalled();
  });
});
