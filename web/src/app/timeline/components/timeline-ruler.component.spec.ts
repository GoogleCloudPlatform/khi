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
import { TimeRangeNs } from './interaction-model';

describe('TimelineRulerComponent', () => {
  let fixture: ComponentFixture<TimelineRulerComponent>;

  const mockViewModel: TimelineRulerViewModel = {
    tickTimeNs: 1000n,
    histogramBucketTimeNs: 1000n,
    histogramBeginTimeNs: 0n,
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
    fixture.componentRef.setInput('leftEdgeTime', 1000n);
    fixture.componentRef.setInput('pixelsPerNs', 2);
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
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1500n,
      endNs: 2000n,
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
    fixture.componentRef.setInput('previewTimeRangeNs', {
      startNs: 1100n,
      endNs: 1200n,
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

  it('should update previewRangeStyle, emit previewTimeRangeNs, and emit timeRangeSelected on mouseup', () => {
    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let previewRange: TimeRangeNs | null | undefined;
    fixture.componentInstance.previewTimeRangeNs.subscribe((range) => {
      previewRange = range;
    });

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    container.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 200,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 400,
      }),
    );
    fixture.detectChanges();

    expect(previewRange).toEqual({ startNs: 1050n, endNs: 1150n });
    const previewOverlay = fixture.nativeElement.querySelector(
      '.time-range-ruler-overlay.preview-range',
    );
    expect(previewOverlay).not.toBeNull();
    expect(previewOverlay.style.getPropertyValue('--range-left')).toBe('100px');
    expect(previewOverlay.style.getPropertyValue('--range-width')).toBe(
      '200px',
    );

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 400,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toEqual({ startNs: 1050n, endNs: 1150n });
    expect(previewRange).toBeNull();
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.preview-range',
      ),
    ).toBeNull();
  });

  it('should normalize startNs <= endNs when dragging from right to left', () => {
    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    container.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 400,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 200,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 200,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toEqual({ startNs: 1050n, endNs: 1150n });
  });

  it('should cancel drag selection and not emit timeRangeSelected when Escape is pressed', () => {
    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let previewRange: TimeRangeNs | null | undefined;
    fixture.componentInstance.previewTimeRangeNs.subscribe((range) => {
      previewRange = range;
    });

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    container.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 200,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 400,
      }),
    );
    fixture.detectChanges();

    expect(previewRange).toEqual({ startNs: 1050n, endNs: 1150n });

    window.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Escape',
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    expect(previewRange).toBeNull();
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.preview-range',
      ),
    ).toBeNull();
    expect(selectedRange).toBeUndefined();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 400,
      }),
    );
    fixture.detectChanges();
    expect(selectedRange).toBeUndefined();
  });

  it('should not emit timeRangeSelected when clicking without dragging (distance < 3px)', () => {
    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    container.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 200,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 201,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toBeUndefined();
  });

  it('should adjust start edge when dragging .range-resize-handle.start while preserving endNs', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let previewRange: TimeRangeNs | null | undefined;
    fixture.componentInstance.previewTimeRangeNs.subscribe((range) => {
      previewRange = range;
    });

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    const startHandle = fixture.nativeElement.querySelector(
      '.range-resize-handle.start',
    ) as HTMLElement;
    expect(startHandle).not.toBeNull();

    startHandle.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 500,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    // clientX = 300 -> relativeX = 200 -> ns = 1000 + 200 / 2 = 1100
    // anchor is endNs = 1600
    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 300,
      }),
    );
    fixture.detectChanges();

    expect(previewRange).toEqual({ startNs: 1100n, endNs: 1600n });

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 300,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toEqual({ startNs: 1100n, endNs: 1600n });
  });

  it('should adjust end edge when dragging .range-resize-handle.end while preserving startNs', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    const endHandle = fixture.nativeElement.querySelector(
      '.range-resize-handle.end',
    ) as HTMLElement;
    expect(endHandle).not.toBeNull();

    endHandle.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 1300,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    // clientX = 1500 -> relativeX = 1400 -> ns = 1000 + 1400 / 2 = 1700
    // anchor is startNs = 1200
    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 1500,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 1500,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toEqual({ startNs: 1200n, endNs: 1700n });
  });

  it('should hide active range overlay during edge resize drag and restore it after mouseup', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range',
      ),
    ).not.toBeNull();

    const startHandle = fixture.nativeElement.querySelector(
      '.range-resize-handle.start',
    ) as HTMLElement;

    startHandle.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 500,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    // The committed range is visually dimmed while resizing an edge, but the
    // overlay stays mounted so the resize handles keep receiving the drag.
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range.is-resizing',
      ),
    ).not.toBeNull();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 500,
      }),
    );
    fixture.detectChanges();

    // Restored after mouseup
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range.is-resizing',
      ),
    ).toBeNull();
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range',
      ),
    ).not.toBeNull();
  });

  it('should emit timeRangeCleared when clicking .range-clear-button without emitting timeRangeSelected or starting a drag', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    let clearedEmitted = false;
    fixture.componentInstance.timeRangeCleared.subscribe(() => {
      clearedEmitted = true;
    });

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    let previewRange: TimeRangeNs | null | undefined;
    fixture.componentInstance.previewTimeRangeNs.subscribe((range) => {
      previewRange = range;
    });

    const clearButton = fixture.nativeElement.querySelector(
      '.range-clear-button',
    ) as HTMLButtonElement;
    expect(clearButton).not.toBeNull();

    clearButton.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 500,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    expect(previewRange).toBeUndefined();

    clearButton.dispatchEvent(
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    expect(clearedEmitted).toBeTrue();
    expect(selectedRange).toBeUndefined();
  });

  it('should invert bounds when the start handle is dragged past the end edge', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    const startHandle = fixture.nativeElement.querySelector(
      '.range-resize-handle.start',
    ) as HTMLElement;
    expect(startHandle).not.toBeNull();

    startHandle.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 500,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    // clientX = 1500 -> relativeX = 1400 -> ns = 1000 + 1400 / 2 = 1700
    // anchor is endNs = 1600, so dragged past anchor
    window.dispatchEvent(
      new MouseEvent('mousemove', {
        clientX: 1500,
      }),
    );
    fixture.detectChanges();

    window.dispatchEvent(
      new MouseEvent('mouseup', {
        clientX: 1500,
      }),
    );
    fixture.detectChanges();

    expect(selectedRange).toEqual({ startNs: 1600n, endNs: 1700n });
  });

  it('should cancel an edge resize and restore the active range overlay when Escape is pressed', () => {
    fixture.componentRef.setInput('activeTimeRangeNs', {
      startNs: 1200n,
      endNs: 1600n,
    });
    fixture.detectChanges();

    const container = fixture.nativeElement.querySelector(
      '.container',
    ) as HTMLElement;
    spyOn(container, 'getBoundingClientRect').and.returnValue({
      left: 100,
      top: 0,
      width: 800,
      height: 60,
    } as DOMRect);

    let selectedRange: TimeRangeNs | undefined;
    fixture.componentInstance.timeRangeSelected.subscribe((range) => {
      selectedRange = range;
    });

    const startHandle = fixture.nativeElement.querySelector(
      '.range-resize-handle.start',
    ) as HTMLElement;
    expect(startHandle).not.toBeNull();

    startHandle.dispatchEvent(
      new MouseEvent('mousedown', {
        button: 0,
        clientX: 500,
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range.is-resizing',
      ),
    ).not.toBeNull();

    window.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'Escape',
        bubbles: true,
        cancelable: true,
      }),
    );
    fixture.detectChanges();

    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range.is-resizing',
      ),
    ).toBeNull();
    expect(
      fixture.nativeElement.querySelector(
        '.time-range-ruler-overlay.active-range',
      ),
    ).not.toBeNull();
    expect(selectedRange).toBeUndefined();
  });
});
