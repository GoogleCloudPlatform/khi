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
import { ChipSearchBarComponent } from './chip-search-bar.component';

describe('ChipSearchBarComponent', () => {
  let component: ChipSearchBarComponent;
  let fixture: ComponentFixture<ChipSearchBarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ChipSearchBarComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(ChipSearchBarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render default placeholder when no terms exist', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;
    expect(inputEl.placeholder).toBe('Search in log body...');
  });

  it('should render chips and OR separators when searchTerms has items', () => {
    fixture.componentRef.setInput('searchTerms', ['foo', 'bar']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    expect(chipEls.length).toBe(2);
    expect(chipEls[0].nativeElement.textContent).toContain('foo');
    expect(chipEls[1].nativeElement.textContent).toContain('bar');

    const orSeparators = fixture.debugElement.queryAll(By.css('.or-separator'));
    expect(orSeparators.length).toBe(2);
    expect(orSeparators[0].nativeElement.textContent.trim()).toBe('OR');
  });

  it('should commit draft to chips when Enter key is pressed', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    component.onDraftInput('error');
    fixture.detectChanges();

    const event = new KeyboardEvent('keydown', { key: 'Enter' });
    inputEl.dispatchEvent(event);
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['error']);
    expect(component.draft()).toBe('');
  });

  it('should commit draft to chips when pipe key is pressed', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    component.onDraftInput('warning');
    fixture.detectChanges();

    const event = new KeyboardEvent('keydown', { key: '|' });
    inputEl.dispatchEvent(event);
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['warning']);
    expect(component.draft()).toBe('');
  });

  it('should commit draft to chips on input blur', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    component.onDraftInput('timeout');
    fixture.detectChanges();

    inputEl.dispatchEvent(new Event('blur'));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['timeout']);
    expect(component.draft()).toBe('');
  });

  it('should remove a chip when its remove button is clicked', () => {
    fixture.componentRef.setInput('searchTerms', ['foo', 'bar']);
    fixture.detectChanges();

    const removeBtns = fixture.debugElement.queryAll(
      By.css('.remove-chip-btn'),
    );
    expect(removeBtns.length).toBe(2);

    removeBtns[0].nativeElement.click();
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['bar']);
  });

  it('should pop last chip into draft on Backspace when draft is empty', () => {
    fixture.componentRef.setInput('searchTerms', ['first', 'second']);
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    const event = new KeyboardEvent('keydown', { key: 'Backspace' });
    inputEl.dispatchEvent(event);
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['first']);
    expect(component.draft()).toBe('second');
  });

  it('should clear draft on Escape, and clear all if draft is empty', () => {
    fixture.componentRef.setInput('searchTerms', ['term1']);
    component.onDraftInput('term2');
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    // First Escape clears draft
    inputEl.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    fixture.detectChanges();
    expect(component.draft()).toBe('');
    expect(component.searchTerms()).toEqual(['term1']);

    // Second Escape clears all chips
    inputEl.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    fixture.detectChanges();
    expect(component.searchTerms()).toEqual([]);
  });

  it('should split pasted text with pipe delimiters into multiple chips', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    const clipboardData = new DataTransfer();
    clipboardData.setData('text/plain', 'alpha | beta|gamma\ndelta');
    const pasteEvent = new ClipboardEvent('paste', {
      clipboardData,
      bubbles: true,
      cancelable: true,
    });

    inputEl.dispatchEvent(pasteEvent);
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual([
      'alpha',
      'beta',
      'gamma',
      'delta',
    ]);
  });

  it('should combine draft at selection when pasting delimited text', () => {
    const inputEl = fixture.debugElement.query(By.css('.search-text-input'))
      .nativeElement as HTMLInputElement;

    component.onDraftInput('prefix_suffix');
    fixture.detectChanges();

    inputEl.setSelectionRange(7, 7); // between 'prefix_' and 'suffix'

    const clipboardData = new DataTransfer();
    clipboardData.setData('text/plain', 'middle1 | middle2');
    const pasteEvent = new ClipboardEvent('paste', {
      clipboardData,
      bubbles: true,
      cancelable: true,
    });

    inputEl.dispatchEvent(pasteEvent);
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual([
      'prefix_middle1',
      'middle2suffix',
    ]);
    expect(component.draft()).toBe('');
  });

  it('should prevent default on mousedown for remove button to preserve input focus', () => {
    fixture.componentRef.setInput('searchTerms', ['chip1']);
    fixture.detectChanges();

    const removeBtn = fixture.debugElement.query(By.css('.remove-chip-btn'));
    const mousedownEvent = new MouseEvent('mousedown', {
      bubbles: true,
      cancelable: true,
    });

    removeBtn.nativeElement.dispatchEvent(mousedownEvent);

    expect(mousedownEvent.defaultPrevented).toBeTrue();
  });

  it('should prevent default on mousedown for clear button to preserve input focus', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const clearBtn = fixture.debugElement.query(By.css('.clear-search-btn'));
    const mousedownEvent = new MouseEvent('mousedown', {
      bubbles: true,
      cancelable: true,
    });

    clearBtn.nativeElement.dispatchEvent(mousedownEvent);

    expect(mousedownEvent.defaultPrevented).toBeTrue();
  });

  it('should clear all chips and draft when clicking clear button', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const clearBtn = fixture.debugElement.query(By.css('.clear-search-btn'));
    expect(clearBtn).toBeTruthy();

    clearBtn.nativeElement.click();
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual([]);
    expect(component.draft()).toBe('');
  });
});
