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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogModule } from '@angular/material/dialog';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressBarHarness } from '@angular/material/progress-bar/testing';

import { ProgressDialogComponent } from './progress.component';
import {
  CurrentProgress,
  PROGRESS_DIALOG_STATUS_OBSERVER,
  ProgressDialogStatusObserver,
} from 'src/app/services/progress/progress-interface';
import { Subject } from 'rxjs';
import { By } from '@angular/platform-browser';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';

describe('ProgressOverlayComponent', () => {
  let component: ProgressDialogComponent;
  let fixture: ComponentFixture<ProgressDialogComponent>;
  let progressObserverSpy: jasmine.SpyObj<ProgressDialogStatusObserver>;
  let progressObserverStatus: Subject<CurrentProgress>;
  let loader: HarnessLoader;

  beforeEach(async () => {
    progressObserverSpy = jasmine.createSpyObj<ProgressDialogStatusObserver>(
      'ProgressDialogStatusObserver',
      ['status'],
    );
    progressObserverStatus = new Subject();
    progressObserverSpy.status.and.returnValue(progressObserverStatus);
    await TestBed.configureTestingModule({
      declarations: [ProgressDialogComponent],
      imports: [MatDialogModule, MatProgressBarModule],
      providers: [
        {
          provide: PROGRESS_DIALOG_STATUS_OBSERVER,
          useValue: progressObserverSpy,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ProgressDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    loader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should show the message obtained from ProgressDialogStatusObserver', () => {
    progressObserverStatus.next({
      message: 'foo',
      percent: 10,
      mode: 'determinate',
    });
    fixture.detectChanges();

    expect(
      fixture.debugElement.query(By.css('.progress-detail')).nativeElement
        .innerText,
    ).toBe('foo');
  });

  it('should update percentage from ProgressDialogStatusObserver', async () => {
    progressObserverStatus.next({
      message: 'foo',
      percent: 10,
      mode: 'determinate',
    });
    fixture.detectChanges();
    const matProgress = await loader.getAllHarnesses(MatProgressBarHarness);

    expect(await matProgress[0].getValue()).toBe(10);
  });

  it('should update mode from ProgressDialogStatusObserver', async () => {
    progressObserverStatus.next({
      message: 'foo',
      percent: 10,
      mode: 'determinate',
    });
    fixture.detectChanges();
    const matProgress = await loader.getAllHarnesses(MatProgressBarHarness);

    expect(await matProgress[0].getMode()).toBe('determinate');

    progressObserverStatus.next({
      message: 'foo',
      percent: 10,
      mode: 'indeterminate',
    });
    fixture.detectChanges();
    expect(await matProgress[0].getMode()).toBe('indeterminate');
  });
});
