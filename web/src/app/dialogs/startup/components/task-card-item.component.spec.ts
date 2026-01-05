/**
 * Copyright 2025 Google LLC
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
import {
  TaskCardItemComponent,
  TaskCardItemViewModel,
} from './task-card-item.component';
import { MatIconTestingModule } from '@angular/material/icon/testing';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { By } from '@angular/platform-browser';
import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';

describe('TaskCardItem', () => {
  let component: TaskCardItemComponent;
  let fixture: ComponentFixture<TaskCardItemComponent>;

  const mockTask: TaskCardItemViewModel = {
    id: 'test-task',
    inspectionTimeLabel: 'now',
    label: 'Test Task',
    phase: 'RUNNING',
    totalProgress: {
      id: 'total',
      label: 'Progress',
      message: 'Processing',
      percentage: 50,
      percentageLabel: '50',
      indeterminate: false,
    },
    progresses: [],
    errors: [],
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        TaskCardItemComponent,
        MatIconTestingModule,
        MatButtonModule,
        MatTooltipModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TaskCardItemComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('task', mockTask);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should switch to edit mode when title is clicked', () => {
    const titleEl = fixture.debugElement.query(By.css('.title'));
    titleEl.nativeElement.click();
    fixture.detectChanges();
    expect(component.isEditing()).toBeTrue();
    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    expect(inputEl).toBeTruthy();
  });

  it('should commit title change on enter', () => {
    component.startEditing();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'New Title';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.triggerEventHandler('keydown.enter', {});

    expect(component.isEditing()).toBeFalse();
    expect(component.changeInspectionTitle.emit).toHaveBeenCalledWith({
      id: 'test-task',
      changeTo: 'New Title',
    });
  });

  it('should cancel editing on escape', () => {
    component.startEditing();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'New Title';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.triggerEventHandler('keydown.escape', {});

    expect(component.isEditing()).toBeFalse();
    expect(component.changeInspectionTitle.emit).not.toHaveBeenCalled();
  });

  function checkButtonVisibilityForPhase(
    phase: InspectionMetadataProgressPhase,
    buttonClass: string,
    wantVisible: boolean,
  ) {
    it(`should ${wantVisible ? 'show' : 'not show'} ${buttonClass} on phase=${phase}`, () => {
      const task2 = {
        ...mockTask,
      };
      task2.phase = phase;
      fixture.componentRef.setInput('task', task2);
      fixture.detectChanges();
      const openBtn = fixture.debugElement.query(By.css(buttonClass));
      if (wantVisible) {
        expect(openBtn).not.toBeNull();
      } else {
        expect(openBtn).toBeNull();
      }
    });
  }

  checkButtonVisibilityForPhase('RUNNING', '.open-button', false);
  checkButtonVisibilityForPhase('DONE', '.open-button', true);
  checkButtonVisibilityForPhase('ERROR', '.open-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.open-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.metadata-button', false);
  checkButtonVisibilityForPhase('DONE', '.metadata-button', true);
  checkButtonVisibilityForPhase('ERROR', '.metadata-button', true);
  checkButtonVisibilityForPhase('CANCELLED', '.metadata-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.cancel-button', true);
  checkButtonVisibilityForPhase('DONE', '.cancel-button', false);
  checkButtonVisibilityForPhase('ERROR', '.cancel-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.cancel-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.download-button', false);
  checkButtonVisibilityForPhase('DONE', '.download-button', true);
  checkButtonVisibilityForPhase('ERROR', '.download-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.download-button', false);
});
