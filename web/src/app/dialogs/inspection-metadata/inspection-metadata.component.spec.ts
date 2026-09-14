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
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { InspectionMetadataOfRunResult } from 'src/app/common/schema/api-types';
import { InspectionMetadataDialogComponent } from './inspection-metadata.component';

describe('InspectionMetadataDialogComponent', () => {
  let component: InspectionMetadataDialogComponent;
  let fixture: ComponentFixture<InspectionMetadataDialogComponent>;
  let mockDialogRef: jasmine.SpyObj<
    MatDialogRef<InspectionMetadataDialogComponent>
  >;

  const mockMetadata: InspectionMetadataOfRunResult = {
    header: {
      inspectionType: 'gcp-gke',
      inspectionName: 'Cluster Alpha',
      inspectionTypeIconPath: '',
      startTimeUnixSeconds: 1700000000,
      endTimeUnixSeconds: 1700003600,
      inspectTimeUnixSeconds: 1700003700,
      suggestedFilename: 'cluster-alpha.khi',
      fileSize: 1048576,
    },
    query: [
      {
        id: 'q1',
        name: 'Audit query',
        query: 'resource.type="k8s_cluster"',
        estimatedCount: 50,
      },
    ],
    log: [
      {
        id: 'l1',
        name: 'AuditLogFetcherTask',
        log: 'Logs fetched.',
      },
    ],
    plan: {
      taskGraph: 'digraph G {}',
    },
    error: {
      errorMessages: [
        {
          errorId: 'ERR_1',
          message: 'An error occurred',
          link: 'https://example.com',
        },
      ],
    },
  };

  beforeEach(async () => {
    mockDialogRef = jasmine.createSpyObj('MatDialogRef', ['close']);

    await TestBed.configureTestingModule({
      imports: [InspectionMetadataDialogComponent, NoopAnimationsModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: mockMetadata,
        },
        {
          provide: MatDialogRef,
          useValue: mockDialogRef,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionMetadataDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and compute view model from dialog data', () => {
    expect(component).toBeTruthy();
    const vm = component.vm();
    expect(vm.overview.inspectionName).toBe('Cluster Alpha');
    expect(vm.overview.inspectionType).toBe('gcp-gke');
    expect(vm.queries.length).toBe(1);
    expect(vm.logs.length).toBe(1);
    expect(vm.errors.length).toBe(1);
    expect(vm.plan.taskGraph).toBe('digraph G {}');
  });

  it('should close dialog when close is invoked', () => {
    component.close();
    expect(mockDialogRef.close).toHaveBeenCalled();
  });

  it('should render the layout component', () => {
    const layoutEl = fixture.nativeElement.querySelector(
      'khi-inspection-metadata-layout',
    );
    expect(layoutEl).toBeTruthy();
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
