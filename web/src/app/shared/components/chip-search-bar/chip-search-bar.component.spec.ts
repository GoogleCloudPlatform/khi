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

  it('should switch chip to editing mode when clicked', () => {
    fixture.componentRef.setInput('searchTerms', ['foo', 'bar']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[0].nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(0);
    expect(component.editingText()).toBe('foo');

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'));
    expect(editInput).toBeTruthy();
    expect((editInput.nativeElement as HTMLInputElement).value).toBe('foo');
  });

  it('should commit edited chip on Enter key', () => {
    fixture.componentRef.setInput('searchTerms', ['foo', 'bar']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[0].nativeElement.click();
    fixture.detectChanges();

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'))
      .nativeElement as HTMLInputElement;
    component.onChipEditInput('baz');
    fixture.detectChanges();

    editInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['baz', 'bar']);
    expect(component.editingIndex()).toBeNull();
    expect(fixture.debugElement.query(By.css('.chip-edit-input'))).toBeNull();
  });

  it('should cancel editing and restore original text on Escape key', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const chipEl = fixture.debugElement.query(By.css('.search-chip'));
    chipEl.nativeElement.click();
    fixture.detectChanges();

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'))
      .nativeElement as HTMLInputElement;
    component.onChipEditInput('edited');
    fixture.detectChanges();

    editInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['foo']);
    expect(component.editingIndex()).toBeNull();
  });

  it('should commit edited chip on blur', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const chipEl = fixture.debugElement.query(By.css('.search-chip'));
    chipEl.nativeElement.click();
    fixture.detectChanges();

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'))
      .nativeElement as HTMLInputElement;
    component.onChipEditInput('blurred');
    fixture.detectChanges();

    editInput.dispatchEvent(new Event('blur'));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['blurred']);
    expect(component.editingIndex()).toBeNull();
  });

  it('should remove chip if committed value is empty or whitespace only', () => {
    fixture.componentRef.setInput('searchTerms', ['foo', 'bar']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[0].nativeElement.click();
    fixture.detectChanges();

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'))
      .nativeElement as HTMLInputElement;
    component.onChipEditInput('   ');
    fixture.detectChanges();

    editInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['bar']);
    expect(component.editingIndex()).toBeNull();
  });

  it('should split into multiple chips when delimiter is in edited chip text', () => {
    fixture.componentRef.setInput('searchTerms', ['first', 'second']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[0].nativeElement.click();
    fixture.detectChanges();

    const editInput = fixture.debugElement.query(By.css('.chip-edit-input'))
      .nativeElement as HTMLInputElement;
    component.onChipEditInput('alpha | beta\ngamma');
    fixture.detectChanges();

    editInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual([
      'alpha',
      'beta',
      'gamma',
      'second',
    ]);
    expect(component.editingIndex()).toBeNull();
  });

  it('should remove chip and exit edit mode when remove button is clicked during edit', () => {
    fixture.componentRef.setInput('searchTerms', ['chip1', 'chip2']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[0].nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(0);

    const removeBtn = fixture.debugElement.query(By.css('.remove-chip-btn'));
    removeBtn.nativeElement.click();
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['chip2']);
    expect(component.editingIndex()).toBeNull();
  });

  it('should adjust editingIndex when a preceding chip is removed', () => {
    fixture.componentRef.setInput('searchTerms', ['chip1', 'chip2', 'chip3']);
    fixture.detectChanges();

    const chipEls = fixture.debugElement.queryAll(By.css('.search-chip'));
    chipEls[1].nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(1);

    const removeBtns = fixture.debugElement.queryAll(
      By.css('.remove-chip-btn'),
    );
    removeBtns[0].nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(0);
    expect(component.searchTerms()).toEqual(['chip2', 'chip3']);
  });

  it('should clear editing state when clear button is clicked during edit', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const chipEl = fixture.debugElement.query(By.css('.search-chip'));
    chipEl.nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(0);

    const clearBtn = fixture.debugElement.query(By.css('.clear-search-btn'));
    clearBtn.nativeElement.click();
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual([]);
    expect(component.editingIndex()).toBeNull();
    expect(component.editingText()).toBe('');
  });

  it('should commit main draft before starting chip edit', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    component.onDraftInput('draft_term');
    fixture.detectChanges();

    const chipEl = fixture.debugElement.query(By.css('.search-chip'));
    chipEl.nativeElement.click();
    fixture.detectChanges();

    expect(component.searchTerms()).toEqual(['foo', 'draft_term']);
    expect(component.draft()).toBe('');
    expect(component.editingIndex()).toBe(0);
  });

  it('should not close edit mode when clicking inside the active chip editor', () => {
    fixture.componentRef.setInput('searchTerms', ['foo']);
    fixture.detectChanges();

    const chipEl = fixture.debugElement.query(By.css('.search-chip'));
    chipEl.nativeElement.click();
    fixture.detectChanges();

    expect(component.editingIndex()).toBe(0);

    const editingChipDiv = fixture.debugElement.query(
      By.css('.search-chip.editing'),
    );
    const clickEvent = new MouseEvent('click', { bubbles: true });
    spyOn(clickEvent, 'stopPropagation');
    editingChipDiv.nativeElement.dispatchEvent(clickEvent);

    expect(clickEvent.stopPropagation).toHaveBeenCalled();
    expect(component.editingIndex()).toBe(0);
  });

  it('should adjust index and handle bounds check in startChipEdit', () => {
    fixture.componentRef.setInput('searchTerms', ['first', 'second', 'third']);
    fixture.detectChanges();

    // Start editing 'first'
    component.startChipEdit(0);
    expect(component.editingIndex()).toBe(0);

    // Empty 'first' so it gets removed, then start editing 'third' (originally index 2)
    component.onChipEditInput('   ');
    component.startChipEdit(2);

    // Since 'first' was removed, previous index 2 is now index 1 ('third')
    expect(component.searchTerms()).toEqual(['second', 'third']);
    expect(component.editingIndex()).toBe(1);
    expect(component.editingText()).toBe('third');

    // Out of bounds startChipEdit should be ignored
    component.startChipEdit(99);
    expect(component.editingIndex()).toBe(1);

    component.startChipEdit(-1);
    expect(component.editingIndex()).toBe(1);
  });
});
