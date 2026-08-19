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
import { ComponentRef } from '@angular/core';
import { IndexProgressLayoutComponent } from './index-progress-layout.component';

describe('IndexProgressLayoutComponent', () => {
  let component: IndexProgressLayoutComponent;
  let componentRef: ComponentRef<IndexProgressLayoutComponent>;
  let fixture: ComponentFixture<IndexProgressLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [IndexProgressLayoutComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(IndexProgressLayoutComponent);
    component = fixture.componentInstance;
    componentRef = fixture.componentRef;
    componentRef.setInput('visible', false);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should not display card when visible is false', () => {
    const card = fixture.debugElement.query(By.css('.index-progress-card'));
    expect(card).toBeNull();
  });

  it('should display card with details when visible is true', () => {
    componentRef.setInput('visible', true);
    componentRef.setInput('percent', 55);
    componentRef.setInput('message', 'Building posting lists...');
    componentRef.setInput('isReady', false);
    fixture.detectChanges();

    const card = fixture.debugElement.query(By.css('.index-progress-card'));
    expect(card).not.toBeNull();

    const titleEl = fixture.debugElement.query(By.css('.title'));
    expect(titleEl.nativeElement.innerText).toBe('Building Search Index');

    const percentEl = fixture.debugElement.query(By.css('.percentage'));
    expect(percentEl.nativeElement.innerText).toContain('55%');

    const messageEl = fixture.debugElement.query(By.css('.message'));
    expect(messageEl.nativeElement.innerText).toBe('Building posting lists...');
  });

  it('should display ready state when isReady is true', () => {
    componentRef.setInput('visible', true);
    componentRef.setInput('percent', 100);
    componentRef.setInput('message', 'Search index ready.');
    componentRef.setInput('isReady', true);
    fixture.detectChanges();

    const titleEl = fixture.debugElement.query(By.css('.title'));
    expect(titleEl.nativeElement.innerText).toBe('Search Index Ready');

    const statusIcon = fixture.debugElement.query(By.css('.status-icon'));
    expect(statusIcon.classes['ready']).toBeTrue();
  });
});
