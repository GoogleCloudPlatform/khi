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
import { MetadataOverviewComponent } from './metadata-overview.component';
import { MetadataOverviewViewModel } from '../types/inspection-metadata.model';

describe('MetadataOverviewComponent', () => {
  let component: MetadataOverviewComponent;
  let fixture: ComponentFixture<MetadataOverviewComponent>;

  const mockOverview: MetadataOverviewViewModel = {
    inspectionType: 'gcp-gke',
    inspectionName: 'Test Cluster Inspection',
    inspectionTypeIconPath: '',
    startTimeUnixSeconds: 1700000000,
    endTimeUnixSeconds: 1700003600,
    inspectTimeUnixSeconds: 1700000000,
    formattedStartTime: '2023/11/14 22:13:20',
    formattedEndTime: '2023/11/14 23:13:20',
    durationText: '1h',
    suggestedFilename: 'cluster.khi',
    fileSizeText: '4.2 MB',
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataOverviewComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MetadataOverviewComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('overview', mockOverview);
    fixture.detectChanges();
  });

  it('should create and render overview items', () => {
    expect(component).toBeTruthy();
    const element = fixture.nativeElement;
    expect(element.textContent).toContain('Test Cluster Inspection');
    expect(element.textContent).toContain('gcp-gke');
    expect(element.textContent).toContain('1h');
    expect(element.textContent).toContain('cluster.khi');
    expect(element.textContent).toContain('4.2 MB');
  });

  it('should render icon image when icon path is present', () => {
    fixture.componentRef.setInput('overview', {
      ...mockOverview,
      inspectionTypeIconPath: 'assets/gke.svg',
    });
    fixture.detectChanges();

    const img = fixture.nativeElement.querySelector('img.type-icon');
    expect(img).toBeTruthy();
    expect(img.getAttribute('src')).toBe('assets/gke.svg');
  });
});
