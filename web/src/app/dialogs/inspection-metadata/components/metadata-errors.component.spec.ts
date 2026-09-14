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
import { MetadataErrorsComponent } from './metadata-errors.component';

describe('MetadataErrorsComponent', () => {
  let component: MetadataErrorsComponent;
  let fixture: ComponentFixture<MetadataErrorsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataErrorsComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MetadataErrorsComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('errors', [
      {
        errorId: 'ERR_PERMISSION',
        message: 'Permission denied on BigQuery API',
        link: 'https://cloud.google.com/error',
      },
    ]);
    fixture.detectChanges();
  });

  it('should create and render error details', () => {
    expect(component).toBeTruthy();
    const el = fixture.nativeElement;
    expect(el.textContent).toContain('ERR_PERMISSION');
    expect(el.textContent).toContain('Permission denied on BigQuery API');

    const link = el.querySelector('a.error-link');
    expect(link).toBeTruthy();
    expect(link.getAttribute('href')).toBe('https://cloud.google.com/error');
  });

  it('should handle errors without links gracefully', () => {
    fixture.componentRef.setInput('errors', [
      {
        errorId: 'ERR_TIMEOUT',
        message: 'Task timed out after 30s',
        link: '',
      },
    ]);
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent).toContain('ERR_TIMEOUT');
    expect(el.textContent).toContain('Task timed out after 30s');
    const link = el.querySelector('a.error-link');
    expect(link).toBeNull();
  });
});
