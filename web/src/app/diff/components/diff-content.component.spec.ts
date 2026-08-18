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
import { DiffContentComponent } from './diff-content.component';
import { Revision } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('DiffContentComponent', () => {
  let component: DiffContentComponent;
  let fixture: ComponentFixture<DiffContentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiffContentComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffContentComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('currentRevision', null);
    fixture.componentRef.setInput('currentRevisionContent', '');
    fixture.componentRef.setInput('showManagedFields', false);
    fixture.componentRef.setInput('timezoneShift', 0);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render nothing when currentRevision is null', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.diff')).toBeNull();
  });

  it('should render loading overlay when isLoading is true and currentRevision is present', () => {
    const mockRevision = {
      id: 1,
      logIndex: 0,
    } as unknown as ReadonlyDomainElement<Revision>;

    fixture.componentRef.setInput('currentRevision', mockRevision);
    fixture.componentRef.setInput('isLoading', true);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.loading-overlay')).toBeTruthy();
    expect(compiled.querySelector('mat-spinner')).toBeTruthy();
    expect(compiled.textContent).toContain('Loading diff content...');
    expect(compiled.querySelector('khi-yaml-viewer')).toBeNull();
  });

  it('should render yaml viewer when isLoading is false and currentRevision is present', () => {
    const mockRevision = {
      id: 1,
      logIndex: 0,
    } as unknown as ReadonlyDomainElement<Revision>;

    fixture.componentRef.setInput('currentRevision', mockRevision);
    fixture.componentRef.setInput('isLoading', false);
    fixture.componentRef.setInput(
      'currentRevisionContent',
      'metadata:\n  name: pod-1\n',
    );
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.loading-overlay')).toBeNull();
    expect(compiled.querySelector('khi-yaml-viewer')).toBeTruthy();
  });
});
