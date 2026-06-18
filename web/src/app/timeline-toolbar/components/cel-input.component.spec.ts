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
import { By } from '@angular/platform-browser';
import { CelInputComponent } from 'src/app/timeline-toolbar/components/cel-input.component';

describe('CelInputComponent', () => {
  let component: CelInputComponent;
  let fixture: ComponentFixture<CelInputComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(CelInputComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create component', () => {
    expect(component).toBeTruthy();
  });

  it('should not display any validation icon when value is empty', () => {
    fixture.componentRef.setInput('value', '');
    fixture.componentRef.setInput('errorMessage', 'Some error');
    fixture.detectChanges();

    const validIcon = fixture.debugElement.query(By.css('.status-icon.valid'));
    const invalidIcon = fixture.debugElement.query(
      By.css('.status-icon.invalid'),
    );

    expect(validIcon).toBeNull();
    expect(invalidIcon).toBeNull();
  });

  it('should display valid icon when value is present and errorMessage is empty', () => {
    fixture.componentRef.setInput('value', 'timeline.name == "foo"');
    fixture.componentRef.setInput('errorMessage', '');
    fixture.detectChanges();

    const validIcon = fixture.debugElement.query(By.css('.status-icon.valid'));
    const invalidIcon = fixture.debugElement.query(
      By.css('.status-icon.invalid'),
    );

    expect(validIcon).not.toBeNull();
    expect(invalidIcon).toBeNull();
  });

  it('should display invalid icon and bubble when value is present and errorMessage is set', () => {
    fixture.componentRef.setInput('value', 'timeline.name ==');
    fixture.componentRef.setInput(
      'errorMessage',
      'Invalid CEL expression error',
    );
    fixture.detectChanges();

    const validIcon = fixture.debugElement.query(By.css('.status-icon.valid'));
    const invalidIcon = fixture.debugElement.query(
      By.css('.status-icon.invalid'),
    );
    const errorBubble = fixture.debugElement.query(
      By.css('.error-bubble-content'),
    );

    expect(validIcon).toBeNull();
    expect(invalidIcon).not.toBeNull();
    expect(errorBubble.nativeElement.textContent.trim()).toBe(
      'Invalid CEL expression error',
    );
  });
});
