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
import { InspectionListComponent } from './inspection-list.component';
import { By } from '@angular/platform-browser';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';
import { Component, input, output } from '@angular/core';
import { InspectionListItemComponent } from './inspection-list-item.component';

// Mock Component
@Component({
  selector: 'khi-inspection-list-item',
  template: '<div class="mock-item"></div>',
})
class MockInspectionListItemComponent {
  item = input.required<InspectionListItemViewModel>();
  openInspectionResult = output<string>();
  openInspectionMetadata = output<string>();
  cancelInspection = output<string>();
  downloadInspectionResult = output<string>();
  changeInspectionTitle = output<{ id: string; changeTo: string }>();
}

describe('InspectionListComponent', () => {
  let component: InspectionListComponent;
  let fixture: ComponentFixture<InspectionListComponent>;

  const mockItems: InspectionListItemViewModel[] = [
    {
      id: 'task-1',
      label: 'Task 1',
      phase: 'DONE',
      inspectionTimeLabel: '2026-04-06 12:00:00',
      totalProgress: {
        id: 'total-1',
        percentage: 100,
        percentageLabel: '100',
        label: 'Total',
        message: 'Done',
        indeterminate: false,
      },
      progresses: [],
      errors: [],
    },
    {
      id: 'task-2',
      label: 'Task 2',
      phase: 'RUNNING',
      inspectionTimeLabel: '2026-04-06 12:05:00',
      totalProgress: {
        id: 'total-2',
        percentage: 50,
        percentageLabel: '50',
        label: 'Total',
        message: 'Running',
        indeterminate: false,
      },
      progresses: [],
      errors: [],
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InspectionListComponent],
    })
      .overrideComponent(InspectionListComponent, {
        remove: { imports: [InspectionListItemComponent] },
        add: { imports: [MockInspectionListItemComponent] },
      })
      .compileComponents();

    fixture = TestBed.createComponent(InspectionListComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('items', mockItems);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render list items', () => {
    const items = fixture.debugElement.queryAll(
      By.css('khi-inspection-list-item'),
    );
    expect(items.length).toBe(2);
  });

  it('should show empty state when no items', () => {
    fixture.componentRef.setInput('items', []);
    fixture.detectChanges();

    const emptyState = fixture.debugElement.query(By.css('.empty-state'));
    expect(emptyState).toBeTruthy();
    expect(emptyState.nativeElement.textContent).toContain(
      'No recent inspections found.',
    );
  });

  it('should show loading state when isLoading is true', () => {
    fixture.componentRef.setInput('isLoading', true);
    fixture.detectChanges();

    const loadingOverlay = fixture.debugElement.query(
      By.css('.loading-overlay'),
    );
    expect(loadingOverlay).toBeTruthy();
    expect(loadingOverlay.nativeElement.textContent).toContain(
      'Loading inspections...',
    );
  });

  it('should show New Inspection button in empty state when not in viewer mode', () => {
    fixture.componentRef.setInput('items', []);
    fixture.componentRef.setInput('isViewerMode', false);
    fixture.detectChanges();

    const button = fixture.debugElement.query(By.css('.empty-state button'));
    expect(button).toBeTruthy();
    expect(button.nativeElement.textContent).toContain('New Inspection');

    spyOn(component.createNewInspection, 'emit');
    button.nativeElement.click();
    expect(component.createNewInspection.emit).toHaveBeenCalled();
  });

  it('should not show New Inspection button in empty state when in viewer mode', () => {
    fixture.componentRef.setInput('items', []);
    fixture.componentRef.setInput('isViewerMode', true);
    fixture.detectChanges();

    const button = fixture.debugElement.query(By.css('.empty-state button'));
    expect(button).toBeFalsy();
  });

  it('should propagate openInspectionResult event', () => {
    const itemEl = fixture.debugElement.query(
      By.css('khi-inspection-list-item'),
    );
    const mockItemComponent =
      itemEl.componentInstance as MockInspectionListItemComponent;

    spyOn(component.openInspectionResult, 'emit');
    mockItemComponent.openInspectionResult.emit('task-1');
    expect(component.openInspectionResult.emit).toHaveBeenCalledWith('task-1');
  });

  it('should propagate openInspectionMetadata event', () => {
    const mockItemComponent = fixture.debugElement.query(
      By.css('khi-inspection-list-item'),
    ).componentInstance;
    spyOn(component.openInspectionMetadata, 'emit');
    mockItemComponent.openInspectionMetadata.emit('task-1');
    expect(component.openInspectionMetadata.emit).toHaveBeenCalledWith(
      'task-1',
    );
  });

  it('should propagate cancelInspection event', () => {
    const mockItemComponent = fixture.debugElement.query(
      By.css('khi-inspection-list-item'),
    ).componentInstance;
    spyOn(component.cancelInspection, 'emit');
    mockItemComponent.cancelInspection.emit('task-2');
    expect(component.cancelInspection.emit).toHaveBeenCalledWith('task-2');
  });

  it('should propagate downloadInspectionResult event', () => {
    const mockItemComponent = fixture.debugElement.query(
      By.css('khi-inspection-list-item'),
    ).componentInstance;
    spyOn(component.downloadInspectionResult, 'emit');
    mockItemComponent.downloadInspectionResult.emit('task-1');
    expect(component.downloadInspectionResult.emit).toHaveBeenCalledWith(
      'task-1',
    );
  });

  it('should propagate changeInspectionTitle event', () => {
    const mockItemComponent = fixture.debugElement.query(
      By.css('khi-inspection-list-item'),
    ).componentInstance;
    spyOn(component.changeInspectionTitle, 'emit');
    const request = { id: 'task-1', changeTo: 'New Title' };
    mockItemComponent.changeInspectionTitle.emit(request);
    expect(component.changeInspectionTitle.emit).toHaveBeenCalledWith(request);
  });
});
