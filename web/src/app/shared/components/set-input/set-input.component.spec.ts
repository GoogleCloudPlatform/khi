/**
 * Copyright 2025 Google LLC
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
import { SetInputComponent, SetInputItem } from './set-input.component';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatChipInputEvent } from '@angular/material/chips';

describe('SetInputComponent', () => {
  let component: SetInputComponent;
  let fixture: ComponentFixture<SetInputComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SetInputComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(SetInputComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('choices', []);
    fixture.componentRef.setInput('selectedItems', []);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should compute viewSelectedItems correctly', () => {
    const choices: SetInputItem[] = [
      { id: 'A', value: 'ValA' },
      { id: 'B', value: 'ValB' },
    ];
    fixture.componentRef.setInput('choices', choices);
    fixture.componentRef.setInput('selectedItems', ['A', 'C']);
    fixture.detectChanges();

    const items = component.viewSelectedItems();
    expect(items.length).toBe(2);
    expect(items[0]).toEqual(choices[0]); // Existing choice
    expect(items[1]).toEqual({ id: 'C', value: 'C' }); // Transient item
  });

  it('should filter textFieldCandidates based on input and selection', () => {
    const choices: SetInputItem[] = [
      { id: 'Apple', value: 'apple' },
      { id: 'Banana', value: 'banana' },
      { id: 'Cherry', value: 'cherry' },
    ];
    fixture.componentRef.setInput('choices', choices);
    fixture.componentRef.setInput('selectedItems', ['Apple']);
    fixture.detectChanges();

    // Initial state (empty input) -> Should show unselected items
    expect(component.textFieldCandidates().length).toBe(2);
    expect(component.textFieldCandidates().map((c) => c.id)).toEqual([
      'Banana',
      'Cherry',
    ]);

    // Typing 'ban' -> Should match Banana
    component.inputCtrl.setValue('ban');
    fixture.detectChanges();
    expect(component.textFieldCandidates().length).toBe(1);
    expect(component.textFieldCandidates()[0].id).toBe('Banana');
  });

  it('should add item from text when allowed', () => {
    const choices: SetInputItem[] = [{ id: 'A', value: 'A' }];
    fixture.componentRef.setInput('choices', choices);
    fixture.componentRef.setInput('allowCustomValues', true);
    fixture.detectChanges();

    let emitted: string[] = [];
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    // Simulate adding custom value
    const event = {
      value: 'Custom',
      chipInput: { clear: jasmine.createSpy('clear') },
    } as unknown as MatChipInputEvent;

    component.addItemFromText(event);

    expect(emitted).toEqual(['Custom']);
    expect(event.chipInput!.clear).toHaveBeenCalled();
    expect(component.inputCtrl.value).toBe('');
  });

  it('should strictly enforce choices when allowCustomValues is false', () => {
    const choices: SetInputItem[] = [{ id: 'A', value: 'A' }];
    fixture.componentRef.setInput('choices', choices);
    fixture.componentRef.setInput('allowCustomValues', false);
    fixture.detectChanges();

    let emitted: string[] | null = null;
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    // Try adding invalid value
    const event = {
      value: 'Invalid',
      chipInput: { clear: jasmine.createSpy('clear') },
    } as unknown as MatChipInputEvent;

    component.addItemFromText(event);

    expect(emitted).toBeNull(); // Should not match anything
    expect(event.chipInput!.clear).toHaveBeenCalled();
  });

  it('should remove item', () => {
    fixture.componentRef.setInput('selectedItems', ['A', 'B']);
    fixture.detectChanges();

    let emitted: string[] = [];
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    component.removeItem({ id: 'A', value: 'A' });

    expect(emitted).toEqual(['B']);
  });

  it('should add all items', () => {
    const choices: SetInputItem[] = [
      { id: 'A', value: 'A' },
      { id: 'B', value: 'B' },
    ];
    fixture.componentRef.setInput('choices', choices);
    fixture.componentRef.setInput('selectedItems', ['A']);
    fixture.detectChanges();

    let emitted: string[] = [];
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    component.addAll();

    expect(emitted).toEqual(['A', 'B']); // Order might usually be preserved or appended, but set logic implies unique
  });

  it('should remove all items', () => {
    fixture.componentRef.setInput('selectedItems', ['A', 'B']);
    fixture.detectChanges();

    let emitted: string[] = [];
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    component.removeAll();

    expect(emitted).toEqual([]);
  });

  it('should select only one item', () => {
    fixture.componentRef.setInput('selectedItems', ['A', 'B']);
    fixture.detectChanges();

    let emitted: string[] = [];
    component.selectedItemsChange.subscribe((val) => (emitted = val));

    component.selectOnly({ id: 'C', value: 'C' });

    expect(emitted).toEqual(['C']);
  });
});
