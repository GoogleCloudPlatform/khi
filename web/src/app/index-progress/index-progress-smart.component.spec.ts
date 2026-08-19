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

import {
  ComponentFixture,
  TestBed,
  fakeAsync,
  tick,
} from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { signal } from '@angular/core';
import { IndexProgressSmartComponent } from './index-progress-smart.component';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { IndexProgressLayoutComponent } from './components/index-progress-layout.component';

describe('IndexProgressSmartComponent', () => {
  let component: IndexProgressSmartComponent;
  let fixture: ComponentFixture<IndexProgressSmartComponent>;

  let isIndexBuildingSignal: ReturnType<typeof signal<boolean>>;
  let isIndexReadySignal: ReturnType<typeof signal<boolean>>;
  let indexProgressPercentageSignal: ReturnType<typeof signal<number>>;
  let indexMessageSignal: ReturnType<typeof signal<string>>;

  beforeEach(async () => {
    isIndexBuildingSignal = signal<boolean>(false);
    isIndexReadySignal = signal<boolean>(false);
    indexProgressPercentageSignal = signal<number>(0);
    indexMessageSignal = signal<string>('');

    const workbenchClientMock = {
      isIndexBuilding: isIndexBuildingSignal.asReadonly(),
      isIndexReady: isIndexReadySignal.asReadonly(),
      indexProgressPercentage: indexProgressPercentageSignal.asReadonly(),
      indexMessage: indexMessageSignal.asReadonly(),
    };

    await TestBed.configureTestingModule({
      imports: [IndexProgressSmartComponent],
      providers: [
        {
          provide: WorkbenchClientService,
          useValue: workbenchClientMock,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(IndexProgressSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should be hidden initially when index is not building', () => {
    const layout = fixture.debugElement.query(
      By.directive(IndexProgressLayoutComponent),
    ).componentInstance as IndexProgressLayoutComponent;

    expect(layout.visible()).toBeFalse();
  });

  it('should show layout when index starts building', () => {
    isIndexBuildingSignal.set(true);
    indexProgressPercentageSignal.set(30);
    indexMessageSignal.set('Indexing logs...');
    fixture.detectChanges();

    const layout = fixture.debugElement.query(
      By.directive(IndexProgressLayoutComponent),
    ).componentInstance as IndexProgressLayoutComponent;

    expect(layout.visible()).toBeTrue();
    expect(layout.percent()).toBe(30);
    expect(layout.message()).toBe('Indexing logs...');
    expect(layout.isReady()).toBeFalse();
  });

  it('should transition to ready and dismiss after timeout', fakeAsync(() => {
    isIndexBuildingSignal.set(true);
    indexProgressPercentageSignal.set(50);
    fixture.detectChanges();

    const layout = fixture.debugElement.query(
      By.directive(IndexProgressLayoutComponent),
    ).componentInstance as IndexProgressLayoutComponent;

    expect(layout.visible()).toBeTrue();

    // Transition to ready
    isIndexBuildingSignal.set(false);
    isIndexReadySignal.set(true);
    indexProgressPercentageSignal.set(100);
    indexMessageSignal.set('Search index ready.');
    fixture.detectChanges();

    expect(layout.visible()).toBeTrue();
    expect(layout.isReady()).toBeTrue();

    // After 1500ms, visible should become false
    tick(1500);
    fixture.detectChanges();

    expect(layout.visible()).toBeFalse();
  }));
});
