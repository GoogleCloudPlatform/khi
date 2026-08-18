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
import {
  LogContentComponent,
  LogContentViewModel,
} from './log-content.component';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('LogContentComponent', () => {
  let component: LogContentComponent;
  let fixture: ComponentFixture<LogContentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LogContentComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(LogContentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render empty message when vm is null', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.empty-message')).toBeTruthy();
    expect(compiled.textContent).toContain('No log selected');
  });

  it('should render loading overlay when isLoading is true and vm is provided', () => {
    const mockLog = {
      id: 1,
      logIndex: 0,
      timestamp: 1000n,
      severity: null,
      logType: null,
      displaySummary: 'test log',
    } as unknown as ReadonlyDomainElement<Log>;

    const mockVm: LogContentViewModel = {
      logEntry: mockLog,
      logBody: '',
      resourceRefs: [],
    };

    fixture.componentRef.setInput('vm', mockVm);
    fixture.componentRef.setInput('isLoading', true);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.loading-overlay')).toBeTruthy();
    expect(compiled.querySelector('mat-spinner')).toBeTruthy();
    expect(compiled.textContent).toContain('Loading log body...');
    expect(compiled.querySelector('khi-yaml-viewer')).toBeNull();
  });

  it('should render yaml viewer when isLoading is false and vm is provided', () => {
    const mockLog = {
      id: 1,
      logIndex: 0,
      timestamp: 1000n,
      severity: null,
      logType: null,
      displaySummary: 'test log',
    } as unknown as ReadonlyDomainElement<Log>;

    const mockVm: LogContentViewModel = {
      logEntry: mockLog,
      logBody: 'message: hello world\n',
      resourceRefs: [],
    };

    fixture.componentRef.setInput('vm', mockVm);
    fixture.componentRef.setInput('isLoading', false);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.loading-overlay')).toBeNull();
    expect(compiled.querySelector('khi-yaml-viewer')).toBeTruthy();
  });
});
