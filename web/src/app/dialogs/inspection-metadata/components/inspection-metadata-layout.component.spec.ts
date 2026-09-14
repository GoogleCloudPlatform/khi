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
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { InspectionMetadataLayoutComponent } from './inspection-metadata-layout.component';
import { InspectionMetadataViewModel } from '../types/inspection-metadata.model';

describe('InspectionMetadataLayoutComponent', () => {
  let component: InspectionMetadataLayoutComponent;
  let fixture: ComponentFixture<InspectionMetadataLayoutComponent>;

  const mockData: InspectionMetadataViewModel = {
    overview: {
      inspectionType: 'gcp-gke',
      inspectionName: 'Cluster Alpha Inspection',
      inspectionTypeIconPath: '',
      startTimeUnixSeconds: 1700000000,
      endTimeUnixSeconds: 1700003600,
      inspectTimeUnixSeconds: 1700003700,
      formattedStartTime: '2023/11/14 22:13:20',
      formattedEndTime: '2023/11/14 23:13:20',
      durationText: '1h',
      suggestedFilename: 'cluster-alpha.khi',
      fileSizeText: '5.2 MB',
    },
    queries: [
      {
        id: 'q1',
        name: 'Audit logs query',
        query: 'resource.type="k8s_cluster"',
        estimatedCount: 1500,
      },
    ],
    logs: [
      {
        id: 'l1',
        name: 'task-fetcher',
        log: 'Fetched 1500 audit logs successfully.',
      },
    ],
    plan: {
      taskGraph: 'digraph G { task1 -> task2; }',
    },
    errors: [
      {
        errorId: 'ERR_TIMEOUT',
        message: 'Query timed out after 30 seconds',
        link: 'https://cloud.google.com/error-docs',
      },
    ],
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InspectionMetadataLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionMetadataLayoutComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('data', mockData);
    fixture.detectChanges();
  });

  it('should create and render header information', () => {
    expect(component).toBeTruthy();
    const element = fixture.nativeElement;
    expect(element.textContent).toContain('Inspection Metadata');
    expect(element.textContent).toContain('Cluster Alpha Inspection');
    expect(element.textContent).toContain('gcp-gke');
  });

  it('should emit close event when close button is clicked', () => {
    const emitSpy = spyOn(component.closed, 'emit');
    const closeBtn = fixture.nativeElement.querySelector('.close-button');
    expect(closeBtn).toBeTruthy();
    closeBtn.click();
    expect(emitSpy).toHaveBeenCalled();
  });

  it('should render error panel when errors exist', () => {
    const errorPanel = fixture.nativeElement.querySelector(
      '.error-expansion-panel',
    );
    expect(errorPanel).toBeTruthy();
    expect(errorPanel.textContent).toContain('Errors');
    expect(errorPanel.textContent).toContain('Query timed out');
  });

  it('should not render error panel when errors list is empty', () => {
    fixture.componentRef.setInput('data', {
      ...mockData,
      errors: [],
    });
    fixture.detectChanges();

    const errorPanel = fixture.nativeElement.querySelector(
      '.error-expansion-panel',
    );
    expect(errorPanel).toBeNull();
  });

  it('should expand and collapse all panels via toolbar actions', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.toolbar-button');
    expect(buttons.length).toBe(2);

    // Click Expand All
    buttons[0].click();
    fixture.detectChanges();

    // Click Collapse All
    buttons[1].click();
    fixture.detectChanges();
  });
});
