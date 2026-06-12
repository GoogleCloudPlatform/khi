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
import { TimelineFilterBuilderComponent } from './timeline-filter-builder.component';
import { By } from '@angular/platform-browser';
import { TimelineType } from 'src/app/store/domain/style';

const createMockType = (
  label: string,
  color: [number, number, number, number],
): TimelineType => ({
  id: Math.random(),
  label,
  description: `Mock ${label}`,
  icon: 'timeline',
  backgroundColor: { r: color[0], g: color[1], b: color[2], a: color[3] },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  typeChipBackgroundColor: {
    r: color[0],
    g: color[1],
    b: color[2],
    a: color[3],
  },
  visible: true,
  sortPriority: 0,
  height: 24,
});

const MOCK_TIMELINE_TYPES: TimelineType[] = [
  createMockType('K8sResource', [0.25, 0.32, 0.71, 1]),
  createMockType('K8sNamespace', [0.78, 0.18, 0.36, 1]),
  createMockType('Gke', [0.15, 0.68, 0.37, 1]),
];

describe('TimelineFilterBuilderComponent', () => {
  let component: TimelineFilterBuilderComponent;
  let fixture: ComponentFixture<TimelineFilterBuilderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, TimelineFilterBuilderComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineFilterBuilderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have correct defaults', () => {
    expect(component.selectedTimelineType()).toBe('*');
    expect(component.filterMode()).toBe('regex');
    expect(component.regexValue()).toBe('');
    expect(component.selectedCandidates()).toEqual([]);
  });

  it('should disable Add Filter button when input is empty in regex mode', () => {
    const addButton = fixture.debugElement
      .queryAll(By.css('.actions button'))
      .find(
        (btn) => btn.nativeElement.textContent.trim() === 'Add Filter',
      )!.nativeElement;
    expect(addButton.disabled).toBeTrue();

    fixture.componentRef.setInput('regexValue', 'some-pattern');
    fixture.detectChanges();

    expect(addButton.disabled).toBeFalse();
  });

  it('should disable selection toggle if selectedTimelineType is *', () => {
    fixture.componentRef.setInput('selectedTimelineType', '*');
    fixture.detectChanges();

    const selectionToggle = fixture.debugElement.query(
      By.css('mat-button-toggle[value="selection"]'),
    ).nativeElement;
    expect(selectionToggle.querySelector('button').disabled).toBeTrue();
  });

  it('should reset mode to regex and clear selection when timeline type changes to *', async () => {
    fixture.componentRef.setInput('selectedTimelineType', 'K8sResource');
    fixture.componentRef.setInput('filterMode', 'selection');
    fixture.componentRef.setInput('selectedCandidates', ['my-pod']);
    fixture.detectChanges();

    expect(component.filterMode()).toBe('selection');
    expect(component.selectedCandidates()).toEqual(['my-pod']);

    fixture.componentRef.setInput('selectedTimelineType', '*');
    fixture.detectChanges();

    await fixture.whenStable();

    expect(component.filterMode()).toBe('regex');
    expect(component.selectedCandidates()).toEqual([]);
  });

  it('should emit closeButtonClicked when Cancel is clicked', () => {
    let emitted = false;
    component.closeButtonClicked.subscribe(() => (emitted = true));

    const cancelButton = fixture.debugElement
      .queryAll(By.css('button'))
      .find((btn) => btn.nativeElement.textContent.trim() === 'Cancel');
    expect(cancelButton).toBeTruthy();
    cancelButton!.nativeElement.click();

    expect(emitted).toBeTrue();
  });

  it('should emit confirm with correct values in regex mode', () => {
    let confirmData: {
      timelineType: string;
      mode: 'regex' | 'selection';
      value: string;
    } | null = null;
    component.confirm.subscribe((data) => (confirmData = data));

    fixture.componentRef.setInput('selectedTimelineType', 'K8sResource');
    fixture.componentRef.setInput('showDeleteButton', true);
    fixture.componentRef.setInput('filterMode', 'regex');
    fixture.componentRef.setInput('regexValue', 'my-regex-pattern');
    fixture.detectChanges();

    const addButton = fixture.debugElement
      .queryAll(By.css('.actions button'))
      .find((btn) => btn.nativeElement.textContent.trim() === 'Add Filter');
    expect(addButton).toBeTruthy();
    addButton!.nativeElement.click();

    expect(confirmData!).toEqual({
      timelineType: 'K8sResource',
      mode: 'regex',
      value: 'my-regex-pattern',
    });
  });

  it('should emit confirm with correct values in selection mode with single selection', () => {
    let confirmData: {
      timelineType: string;
      mode: 'regex' | 'selection';
      value: string;
    } | null = null;
    component.confirm.subscribe((data) => (confirmData = data));

    fixture.componentRef.setInput('selectedTimelineType', 'K8sResource');
    fixture.componentRef.setInput('filterMode', 'selection');
    fixture.componentRef.setInput('selectedCandidates', ['pod']);
    fixture.detectChanges();

    const addButton = fixture.debugElement
      .queryAll(By.css('.actions button'))
      .find((btn) => btn.nativeElement.textContent.trim() === 'Add Filter');
    expect(addButton).toBeTruthy();
    addButton!.nativeElement.click();

    expect(confirmData!).toEqual({
      timelineType: 'K8sResource',
      mode: 'selection',
      value: 'pod',
    });
  });

  it('should emit confirm with correct pipe-separated values in selection mode with multiple selections', () => {
    let confirmData: {
      timelineType: string;
      mode: 'regex' | 'selection';
      value: string;
    } | null = null;
    component.confirm.subscribe((data) => (confirmData = data));

    fixture.componentRef.setInput('selectedTimelineType', 'K8sResource');
    fixture.componentRef.setInput('filterMode', 'selection');
    fixture.componentRef.setInput('selectedCandidates', ['pod-x', 'pod-y']);
    fixture.detectChanges();

    const addButton = fixture.debugElement
      .queryAll(By.css('.actions button'))
      .find((btn) => btn.nativeElement.textContent.trim() === 'Add Filter');
    expect(addButton).toBeTruthy();
    addButton!.nativeElement.click();

    expect(confirmData!).toEqual({
      timelineType: 'K8sResource',
      mode: 'selection',
      value: 'pod-x|pod-y',
    });
  });

  it('should not display delete button by default', () => {
    const deleteButton = fixture.debugElement.query(By.css('.delete-btn'));
    expect(deleteButton).toBeNull();
  });

  it('should display delete button and emit deleteButtonClicked when clicked', () => {
    let emitted = false;
    component.deleteButtonClicked.subscribe(() => (emitted = true));

    fixture.componentRef.setInput('showDeleteButton', true);
    fixture.detectChanges();

    const deleteButton = fixture.debugElement.query(By.css('.delete-btn'));
    expect(deleteButton).toBeTruthy();

    deleteButton.nativeElement.click();
    fixture.detectChanges();

    expect(emitted).toBeTrue();
  });

  it('should filter timeline types based on autocomplete input query', () => {
    fixture.componentRef.setInput('timelineTypes', MOCK_TIMELINE_TYPES);
    fixture.detectChanges();

    component['onTypeInputChange']('k8s');
    fixture.detectChanges();

    expect(component['filteredTimelineTypes']().map((t) => t.label)).toEqual([
      'K8sResource',
      'K8sNamespace',
    ]);
  });

  it('should update selectedTimelineType when option is selected', () => {
    fixture.componentRef.setInput('timelineTypes', MOCK_TIMELINE_TYPES);
    fixture.detectChanges();

    component['onTimelineTypeSelected']('Gke');
    fixture.detectChanges();

    expect(component.selectedTimelineType()).toBe('Gke');
  });

  it('should fallback to * when user inputs non-existing type', () => {
    fixture.componentRef.setInput('timelineTypes', MOCK_TIMELINE_TYPES);
    fixture.detectChanges();

    component['onTypeInputChange']('UnknownType');
    fixture.detectChanges();

    expect(component.selectedTimelineType()).toBe('*');
  });
});
