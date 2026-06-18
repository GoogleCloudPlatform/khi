/**
 * Copyright 2024 Google LLC
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
import { SearchBarComponent } from 'src/app/shared/components/search-bar/search-bar.component';
import { By } from '@angular/platform-browser';

describe('SearchBarComponent', () => {
  let component: SearchBarComponent;
  let fixture: ComponentFixture<SearchBarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SearchBarComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SearchBarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should emit queryChange on input', () => {
    let emittedQuery = '';
    component.queryChange.subscribe((q) => {
      emittedQuery = q;
    });

    const inputEl = fixture.debugElement.query(By.css('.search-input'))
      .nativeElement as HTMLInputElement;
    inputEl.value = 'test';
    inputEl.dispatchEvent(new Event('input'));

    expect(emittedQuery).toBe('test');
  });

  it('should display No matches when query is non-empty and matchCount is 0', () => {
    fixture.componentRef.setInput('query', 'test');
    fixture.componentRef.setInput('matchCount', 0);
    fixture.detectChanges();

    const countSpan = fixture.debugElement.query(By.css('.match-count'))
      .nativeElement as HTMLElement;
    expect(countSpan.textContent?.trim()).toBe('No matches');
  });
});
