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

import { TestBed } from '@angular/core/testing';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { ProgressDialogService } from './progress-dialog.service';
import { ProgressDialogComponent } from 'src/app/dialogs/progress/progress.component';
import { firstValueFrom, take, toArray } from 'rxjs';
import {
  PROGRESS_DIALOG_STATUS_OBSERVER,
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusObserver,
  ProgressDialogStatusUpdator,
} from './progress-interface';

describe('ProgressDialogService', () => {
  let matDialogSpy: jasmine.SpyObj<MatDialog>;
  let matDialogRefSpy: jasmine.SpyObj<MatDialogRef<ProgressDialogComponent>>;
  let progressDialogObserver: ProgressDialogStatusObserver;
  let progressDialogUpdator: ProgressDialogStatusUpdator;

  beforeEach(async () => {
    matDialogSpy = jasmine.createSpyObj<MatDialog>('MatDialog', ['open']);
    matDialogRefSpy = jasmine.createSpyObj<
      MatDialogRef<ProgressDialogComponent>
    >('MatDialogRef', ['close']);
    matDialogSpy.open.and.returnValue(matDialogRefSpy);
    await TestBed.configureTestingModule({
      providers: [
        {
          provide: MatDialog,
          useValue: matDialogSpy,
        },
        ...ProgressDialogService.providers(),
      ],
    });
    progressDialogObserver = await TestBed.inject(
      PROGRESS_DIALOG_STATUS_OBSERVER,
    );
    progressDialogUpdator = await TestBed.inject(
      PROGRESS_DIALOG_STATUS_UPDATOR,
    );
  });

  it('provides observer and updator from returened value of providers()', () => {
    expect(progressDialogObserver).toBeTruthy();
    expect(progressDialogUpdator).toBeTruthy();
  });

  it('opens dialog when show() called', () => {
    progressDialogUpdator.show();

    expect(matDialogSpy.open.calls.count()).toBe(1);
  });

  it('ignores show() when the previous dialog have not closed', () => {
    progressDialogUpdator.show();
    progressDialogUpdator.show();

    expect(matDialogSpy.open.calls.count()).toBe(1);
  });

  it('closes dialog when dismiss() called', () => {
    progressDialogUpdator.show();
    progressDialogUpdator.dismiss();

    expect(matDialogSpy.open.calls.count()).toBe(1);
    expect(matDialogRefSpy.close.calls.count()).toBe(1);
  });

  it('ignores dismiss() when the previous dialog had 2 or more show() called', () => {
    progressDialogUpdator.show();
    progressDialogUpdator.show();
    progressDialogUpdator.dismiss();

    expect(matDialogSpy.open.calls.count()).toBe(1);
    expect(matDialogRefSpy.close.calls.count()).toBe(0);
  });

  it('emit status when updateProgress is called', async () => {
    progressDialogUpdator.show();
    const statuses = progressDialogObserver.status().pipe(take(1), toArray());
    progressDialogUpdator.updateProgress({
      message: 'foo',
      percent: 10,
      mode: 'determinate',
    });

    const result = await firstValueFrom(statuses);
    expect(result[0]).toEqual({
      message: 'foo',
      percent: 10,
      mode: 'determinate',
    });
  });
});
