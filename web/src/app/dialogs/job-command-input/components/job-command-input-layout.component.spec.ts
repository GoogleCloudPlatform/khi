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
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { JobCommandInputLayoutComponent } from 'src/app/dialogs/job-command-input/components/job-command-input-layout.component';

describe('JobCommandInputLayoutComponent', () => {
  let component: JobCommandInputLayoutComponent;
  let fixture: ComponentFixture<JobCommandInputLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobCommandInputLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(JobCommandInputLayoutComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should disable submit button when command is empty', () => {
    fixture.componentRef.setInput('command', '');
    fixture.detectChanges();

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const submitBtn = buttons.find((b) =>
      b.nativeElement.textContent.trim().includes('Run'),
    );
    expect(submitBtn).toBeTruthy();
    expect(submitBtn!.nativeElement.disabled).toBeTrue();
  });

  it('should enable submit button when command has non-whitespace content', () => {
    fixture.componentRef.setInput('command', './khi --job-mode');
    fixture.detectChanges();

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const submitBtn = buttons.find((b) =>
      b.nativeElement.textContent.trim().includes('Run'),
    );
    expect(submitBtn).toBeTruthy();
    expect(submitBtn!.nativeElement.disabled).toBeFalse();
  });

  it('should update command model signal when user types in textarea', () => {
    fixture.detectChanges();
    const textarea = fixture.debugElement.query(By.css('textarea'))
      .nativeElement as HTMLTextAreaElement;
    textarea.value = './khi --job-mode --job-inspection-type="gke"';
    textarea.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(component.command()).toBe(
      './khi --job-mode --job-inspection-type="gke"',
    );
  });

  it('should render error message when errorMessage is provided', () => {
    fixture.componentRef.setInput('errorMessage', 'Failed to parse command');
    fixture.detectChanges();

    const errorCallout = fixture.debugElement.query(By.css('.error-callout'));
    expect(errorCallout).toBeTruthy();
    expect(errorCallout.nativeElement.textContent).toContain(
      'Failed to parse command',
    );
  });

  it('should emit submitCommand event when submit button is clicked', () => {
    fixture.componentRef.setInput('command', './khi --job-mode');
    fixture.detectChanges();

    let submitEmitted = false;
    component.submitCommand.subscribe(() => {
      submitEmitted = true;
    });

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const submitBtn = buttons.find((b) =>
      b.nativeElement.textContent.trim().includes('Run'),
    );
    submitBtn!.nativeElement.click();

    expect(submitEmitted).toBeTrue();
  });

  it('should emit cancelDialog event when cancel button is clicked', () => {
    fixture.detectChanges();

    let cancelEmitted = false;
    component.cancelDialog.subscribe(() => {
      cancelEmitted = true;
    });

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const cancelBtn = buttons.find((b) =>
      b.nativeElement.textContent.trim().includes('Cancel'),
    );
    cancelBtn!.nativeElement.click();

    expect(cancelEmitted).toBeTrue();
  });
});
