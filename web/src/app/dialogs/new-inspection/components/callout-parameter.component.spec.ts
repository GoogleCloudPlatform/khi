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
import { MatIcon } from '@angular/material/icon';
import { MatProgressSpinner } from '@angular/material/progress-spinner';
import {
  CalloutParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { CalloutParameterComponent } from 'src/app/dialogs/new-inspection/components/callout-parameter.component';

describe('CalloutParameterComponent', () => {
  let fixture: ComponentFixture<CalloutParameterComponent>;

  const baseParameter: CalloutParameterFormField = {
    id: 'log-volume-advisory',
    type: ParameterInputType.Callout,
    label: 'Log Volume Advisory',
    description: 'Applies when querying wide time ranges.',
    hint: 'Inspecting more than 6 hours of logs may increase memory usage.',
    hintType: ParameterHintType.Info,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CalloutParameterComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(CalloutParameterComponent);
  });

  it('renders info severity banner with title, description, message, and info icon without input controls', () => {
    fixture.componentRef.setInput('parameter', baseParameter);
    fixture.detectChanges();

    const card = fixture.debugElement.query(By.css('.callout-card'));
    expect(card.classes['info']).toBeTrue();
    expect(card.classes['warning']).toBeFalsy();
    expect(card.classes['error']).toBeFalsy();

    const icon = fixture.debugElement.query(By.directive(MatIcon));
    expect(icon).toBeTruthy();
    expect(icon.nativeElement.textContent.trim()).toBe('info');

    expect(
      fixture.debugElement.query(By.css('.title')).nativeElement.textContent,
    ).toBe('Log Volume Advisory');
    expect(
      fixture.debugElement.query(By.css('.description')).nativeElement
        .textContent,
    ).toContain('Applies when querying wide time ranges.');
    expect(
      fixture.debugElement.query(By.css('.message')).nativeElement.textContent,
    ).toContain(
      'Inspecting more than 6 hours of logs may increase memory usage.',
    );

    expect(fixture.debugElement.query(By.css('mat-form-field'))).toBeNull();
    expect(fixture.debugElement.query(By.css('input'))).toBeNull();
  });

  it('defaults None or undefined hintType to info severity', () => {
    fixture.componentRef.setInput('parameter', {
      ...baseParameter,
      hintType: ParameterHintType.None,
    });
    fixture.detectChanges();

    let card = fixture.debugElement.query(By.css('.callout-card'));
    expect(card.classes['info']).toBeTrue();
    let icon = fixture.debugElement.query(By.directive(MatIcon));
    expect(icon.nativeElement.textContent.trim()).toBe('info');

    fixture.componentRef.setInput('parameter', {
      ...baseParameter,
      hintType: undefined,
    });
    fixture.detectChanges();

    card = fixture.debugElement.query(By.css('.callout-card'));
    expect(card.classes['info']).toBeTrue();
    icon = fixture.debugElement.query(By.directive(MatIcon));
    expect(icon.nativeElement.textContent.trim()).toBe('info');
  });

  it('renders warning severity banner with warning icon', () => {
    fixture.componentRef.setInput('parameter', {
      ...baseParameter,
      id: 'api-quota-warning',
      label: 'API Quota Warning',
      hint: 'Cloud Logging read quota is below recommended threshold.',
      hintType: ParameterHintType.Warning,
    });
    fixture.detectChanges();

    const card = fixture.debugElement.query(By.css('.callout-card'));
    expect(card.classes['warning']).toBeTrue();
    const icon = fixture.debugElement.query(By.directive(MatIcon));
    expect(icon.nativeElement.textContent.trim()).toBe('warning');
  });

  it('renders error severity banner with error icon', () => {
    fixture.componentRef.setInput('parameter', {
      ...baseParameter,
      id: 'missing-log-bucket',
      label: 'Missing Required Log Bucket',
      hint: 'Required log bucket was not found in project test-project.',
      hintType: ParameterHintType.Error,
    });
    fixture.detectChanges();

    const card = fixture.debugElement.query(By.css('.callout-card'));
    expect(card.classes['error']).toBeTrue();
    const icon = fixture.debugElement.query(By.directive(MatIcon));
    expect(icon.nativeElement.textContent.trim()).toBe('error');
  });

  it('renders progress spinner instead of icon when pending is true', () => {
    fixture.componentRef.setInput('parameter', {
      ...baseParameter,
      pending: true,
    });
    fixture.detectChanges();

    const spinner = fixture.debugElement.query(
      By.directive(MatProgressSpinner),
    );
    expect(spinner).toBeTruthy();
    expect(fixture.debugElement.query(By.directive(MatIcon))).toBeNull();
  });

  it('omits empty optional elements and preserves multiline and angle-bracket text', () => {
    fixture.componentRef.setInput('parameter', {
      id: 'multiline-callout',
      type: ParameterInputType.Callout,
      label: 'Cluster Prerequisites',
      description: '',
      hint: 'Line one.\nSpecify <cluster-name>.',
      hintType: ParameterHintType.Info,
    });
    fixture.detectChanges();

    expect(fixture.debugElement.query(By.css('.description'))).toBeNull();
    const messageEl = fixture.debugElement.query(By.css('.message'))
      .nativeElement as HTMLElement;
    expect(messageEl.textContent).toBe('Line one.\nSpecify <cluster-name>.');
    expect(getComputedStyle(messageEl).whiteSpace).toBe('pre-line');

    fixture.componentRef.setInput('parameter', {
      id: 'minimal-callout',
      type: ParameterInputType.Callout,
      label: '',
      description: 'Scope: <project-id>\nSecondary note.',
      hint: '',
      hintType: ParameterHintType.Info,
    });
    fixture.detectChanges();

    expect(fixture.debugElement.query(By.css('.title'))).toBeNull();
    const descriptionEl = fixture.debugElement.query(By.css('.description'))
      .nativeElement as HTMLElement;
    expect(descriptionEl.textContent).toBe(
      'Scope: <project-id>\nSecondary note.',
    );
    expect(getComputedStyle(descriptionEl).whiteSpace).toBe('pre-line');
    expect(fixture.debugElement.query(By.css('.message'))).toBeNull();
  });
});
