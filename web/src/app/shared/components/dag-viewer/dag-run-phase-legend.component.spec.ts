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
import { DagRunPhaseLegendComponent } from 'src/app/shared/components/dag-viewer/dag-run-phase-legend.component';

describe('DagRunPhaseLegendComponent', () => {
  let fixture: ComponentFixture<DagRunPhaseLegendComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DagRunPhaseLegendComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(DagRunPhaseLegendComponent);
    fixture.detectChanges();
  });

  it('renders one entry per decorated run phase', () => {
    const root = fixture.nativeElement as HTMLElement;
    const items = Array.from(root.querySelectorAll('.legend-item'));

    expect(items.map((item) => item.getAttribute('data-run-phase'))).toEqual([
      'WAITING',
      'RUNNING',
      'DONE',
      'ERROR',
    ]);
    for (const item of items) {
      expect(item.querySelector('.legend-swatch')).toBeTruthy();
      const label = item.querySelector('.legend-label');
      expect(label).toBeTruthy();
      expect(label?.textContent?.trim().length).toBeGreaterThan(0);
    }
  });
});
