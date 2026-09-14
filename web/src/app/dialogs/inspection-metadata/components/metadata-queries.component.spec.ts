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
import { MetadataQueriesComponent } from './metadata-queries.component';

describe('MetadataQueriesComponent', () => {
  let component: MetadataQueriesComponent;
  let fixture: ComponentFixture<MetadataQueriesComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataQueriesComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MetadataQueriesComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('queries', [
      {
        id: 'q1',
        name: 'GKE Audit Query',
        query: 'resource.type="k8s_cluster"',
      },
      {
        id: 'q2',
        name: 'Node System Query',
        query: 'resource.type="gce_instance"',
      },
    ]);
    fixture.detectChanges();
  });

  it('should create and render queries', () => {
    expect(component).toBeTruthy();
    const element = fixture.nativeElement;
    expect(element.textContent).toContain('GKE Audit Query');
    expect(element.textContent).toContain('resource.type="k8s_cluster"');
    expect(element.textContent).toContain('Node System Query');
    expect(element.textContent).toContain('resource.type="gce_instance"');
  });
});
