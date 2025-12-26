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
import { TaskCardListComponent } from './task-card-list.component';
import { TaskCardItemViewModel } from './task-card-item.component';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { By } from '@angular/platform-browser';
import { MatIconTestingModule } from '@angular/material/icon/testing';

describe('TaskCardList', () => {
  let component: TaskCardListComponent;
  let fixture: ComponentFixture<TaskCardListComponent>;

  const mockTasks: TaskCardItemViewModel[] = [
    {
      id: 'task-1',
      inspectionTimeLabel: 'now',
      label: 'Task 1',
      phase: 'RUNNING',
      totalProgress: {
        id: 'p1',
        label: 'P1',
        message: '',
        percentage: 0,
        percentageLabel: '0',
        indeterminate: false,
      },
      progresses: [],
      errors: [],
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        TaskCardListComponent,
        MatProgressBarModule,
        MatIconModule,
        MatIconTestingModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TaskCardListComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('tasks', undefined);
    fixture.componentRef.setInput('isViewerMode', false);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should show loading when tasks is undefined', () => {
    fixture.componentRef.setInput('tasks', undefined);
    fixture.detectChanges();
    const loading = fixture.debugElement.query(By.css('mat-progress-bar'));
    expect(loading).toBeTruthy();
  });

  it('should show empty message when tasks is empty', () => {
    fixture.componentRef.setInput('tasks', []);
    fixture.detectChanges();
    const emptyMsg = fixture.debugElement.query(By.css('.message-container'));
    expect(emptyMsg.nativeElement.textContent).toContain(
      'No inspection tasks found',
    );
  });

  it('should show viewer mode message when isViewerMode is true', () => {
    fixture.componentRef.setInput('isViewerMode', true);
    fixture.detectChanges();
    const viewerMsg = fixture.debugElement.query(By.css('.message-container'));
    expect(viewerMsg.nativeElement.textContent).toContain(
      'KHI is running as viewer mode',
    );
  });

  it('should show tasks when tasks are provided and not viewer mode', () => {
    fixture.componentRef.setInput('tasks', mockTasks);
    fixture.componentRef.setInput('isViewerMode', false);
    fixture.detectChanges();
    const taskItems = fixture.debugElement.queryAll(
      By.css('khi-task-card-item'),
    );
    expect(taskItems.length).toBe(1);
  });
});
