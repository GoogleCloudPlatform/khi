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
import { StartupDialogContentComponent } from './startup-dialog-content.component';
import { Component, input, output } from '@angular/core';
import { By } from '@angular/platform-browser';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';
import { InspectionListComponent } from './inspection-list.component';

@Component({
  selector: 'khi-inspection-list',
  template: '<div></div>',
})
class MockInspectionListComponent {
  public readonly items = input.required<InspectionListItemViewModel[]>();
  public readonly isLoading = input<boolean>(false);
  public readonly isViewerMode = input<boolean>(false);
  public readonly createNewInspection = output<void>();
  public readonly openInspectionResult = output<string>();
  public readonly openInspectionMetadata = output<string>();
  public readonly cancelInspection = output<string>();
  public readonly downloadInspectionResult = output<string>();
  public readonly changeInspectionTitle = output<{
    id: string;
    changeTo: string;
  }>();
}

describe('StartupDialogContentComponent', () => {
  let component: StartupDialogContentComponent;
  let fixture: ComponentFixture<StartupDialogContentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StartupDialogContentComponent],
    })
      .overrideComponent(StartupDialogContentComponent, {
        remove: { imports: [InspectionListComponent] },
        add: { imports: [MockInspectionListComponent] },
      })
      .compileComponents();

    fixture = TestBed.createComponent(StartupDialogContentComponent);
    component = fixture.componentInstance;

    // Provide required inputs
    fixture.componentRef.setInput('items', []);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should pass items to inspection list', () => {
    const testItems: InspectionListItemViewModel[] = [
      {
        id: '1',
        inspectionTimeLabel: '2026-04-06',
        label: 'Test',
        phase: 'RUNNING',
        totalProgress: {
          id: 'total',
          label: 'Total',
          message: 'Running',
          percentage: 50,
          percentageLabel: '50%',
          indeterminate: false,
        },
        progresses: [],
        errors: [],
      },
    ];
    fixture.componentRef.setInput('items', testItems);
    fixture.detectChanges();

    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    expect(list.items()).toEqual(testItems);
  });

  it('should emit openInspectionResult when list emits openInspectionResult', () => {
    const spy = spyOn(component.openInspectionResult, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    list.openInspectionResult.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit openInspectionMetadata when list emits openInspectionMetadata', () => {
    const spy = spyOn(component.openInspectionMetadata, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    list.openInspectionMetadata.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit cancelInspection when list emits cancelInspection', () => {
    const spy = spyOn(component.cancelInspection, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    list.cancelInspection.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit downloadInspectionResult when list emits downloadInspectionResult', () => {
    const spy = spyOn(component.downloadInspectionResult, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    list.downloadInspectionResult.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit changeInspectionTitle when list emits changeInspectionTitle', () => {
    const spy = spyOn(component.changeInspectionTitle, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    const request = { id: 'test-id', changeTo: 'new-title' };
    list.changeInspectionTitle.emit(request);

    expect(spy).toHaveBeenCalledWith(request);
  });

  it('should emit createNewInspection when list emits createNewInspection', () => {
    const spy = spyOn(component.createNewInspection, 'emit');
    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    list.createNewInspection.emit();

    expect(spy).toHaveBeenCalled();
  });

  it('should pass isLoading and isViewerMode to inspection list', () => {
    fixture.componentRef.setInput('isLoading', true);
    fixture.componentRef.setInput('isViewerMode', true);
    fixture.detectChanges();

    const listEl = fixture.debugElement.query(
      By.directive(MockInspectionListComponent),
    );
    const list = listEl.componentInstance as MockInspectionListComponent;

    expect(list.isLoading()).toBeTrue();
    expect(list.isViewerMode()).toBeTrue();
  });
});
