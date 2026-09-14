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
import { MAT_DIALOG_DATA } from '@angular/material/dialog';

import { InspectionMetadataDialogComponent } from './inspection-metadata.component';
import { provideHttpClient } from '@angular/common/http';

describe('TaskMetadataViewDialogComponent', () => {
  let component: InspectionMetadataDialogComponent;
  let fixture: ComponentFixture<InspectionMetadataDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: [],
        },
        provideHttpClient(),
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionMetadataDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render job command when jobCommand is provided', () => {
    const fixtureWithJob = TestBed.createComponent(
      InspectionMetadataDialogComponent,
    );
    const comp = fixtureWithJob.componentInstance;
    (comp as unknown as { data: unknown }).data = {
      jobCommand: { command: './khi --job-mode' },
    };
    fixtureWithJob.detectChanges();
    const compiled = fixtureWithJob.nativeElement as HTMLElement;
    expect(compiled.querySelector('khi-job-command')).toBeTruthy();
  });
});
