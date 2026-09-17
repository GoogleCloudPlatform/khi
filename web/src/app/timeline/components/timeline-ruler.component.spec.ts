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
import { TimelineRulerComponent } from './timeline-ruler.component';
import { TimelineRulerViewModel } from './timeline-ruler.viewmodel';
import { generateDefaultRulerStyle } from 'src/app/timeline/components/style-model';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';

describe('TimelineRulerComponent', () => {
  let fixture: ComponentFixture<TimelineRulerComponent>;

  const mockViewModel: TimelineRulerViewModel = {
    tickTimeMS: 1000,
    histogramBucketTimeMS: 1000,
    histogramBeginTimeMS: 0,
    histogramBuckets: [],
    ticks: [],
    timeLabels: [],
  };

  beforeEach(async () => {
    spyOn(TimelineRulerComponent.prototype, 'ngAfterViewInit').and.stub();
    await TestBed.configureTestingModule({
      imports: [TimelineRulerComponent],
      providers: [
        {
          provide: RenderingLoopManager,
          useValue: {
            addRenderer: jasmine.createSpy('addRenderer'),
            removeRenderer: jasmine.createSpy('removeRenderer'),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineRulerComponent);
    fixture.componentRef.setInput('viewModel', mockViewModel);
    fixture.componentRef.setInput('rulerStyle', generateDefaultRulerStyle());
    fixture.componentRef.setInput('leftEdgeTime', 1000);
    fixture.componentRef.setInput('pixelsPerMs', 2);
    fixture.detectChanges();
  });

  it('should not render overlays when inputs are null', () => {
    const activeOverlay = fixture.nativeElement.querySelector(
      '.time-range-ruler-overlay.active-range',
    );
    const previewOverlay = fixture.nativeElement.querySelector(
      '.time-range-ruler-overlay.preview-range',
    );
    expect(activeOverlay).toBeNull();
    expect(previewOverlay).toBeNull();
  });

  it('should render active range overlay with expected position and width', () => {
    fixture.componentRef.setInput('activeTimeRangeMs', {
      startMs: 1500,
      endMs: 2000,
    });
    fixture.detectChanges();

    const activeOverlay = fixture.nativeElement.querySelector(
      '.time-range-ruler-overlay.active-range',
    );
    expect(activeOverlay).not.toBeNull();
    expect(activeOverlay.style.getPropertyValue('--range-left')).toBe('1000px');
    expect(activeOverlay.style.getPropertyValue('--range-width')).toBe(
      '1000px',
    );
  });

  it('should render preview range overlay with expected position and width', () => {
    fixture.componentRef.setInput('previewTimeRangeMs', {
      startMs: 1100,
      endMs: 1200,
    });
    fixture.detectChanges();

    const previewOverlay = fixture.nativeElement.querySelector(
      '.time-range-ruler-overlay.preview-range',
    );
    expect(previewOverlay).not.toBeNull();
    expect(previewOverlay.style.getPropertyValue('--range-left')).toBe('200px');
    expect(previewOverlay.style.getPropertyValue('--range-width')).toBe(
      '200px',
    );
  });
});
