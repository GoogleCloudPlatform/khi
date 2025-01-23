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

import {
  Component,
  ElementRef,
  EventEmitter,
  Input,
  OnInit,
  Output,
  ViewChild,
} from '@angular/core';
import { FormControl } from '@angular/forms';
import { MatChipInputEvent } from '@angular/material/chips';
import { ENTER, COMMA } from '@angular/cdk/keycodes';
import { MatAutocompleteSelectedEvent } from '@angular/material/autocomplete';
import { map, Observable } from 'rxjs';
import {
  filteElementsByIncludedSubstring,
  iterToArr,
  subtractSet,
} from '../utils/collection-util';
@Component({
  selector: 'khi-header-set-input',
  templateUrl: './set-input.component.html',
  styleUrls: ['./set-input.component.sass'],
})
export class SetInputComponent implements OnInit {
  @Input()
  public label = '';

  @Input()
  public selectedItems: Set<string> = new Set();

  @Input()
  public choices: Set<string> = new Set();

  @Output()
  public selectedItemsChange = new EventEmitter<Set<string>>();

  @Output()
  public closeButtonClicked = new EventEmitter();

  inputCtrl = new FormControl('');

  separatorKeysCodes: number[] = [ENTER, COMMA];

  @ViewChild('inputElement') inputElement!: ElementRef<HTMLInputElement>;

  public $textFieldCandidates!: Observable<string[]>;

  ngOnInit(): void {
    this.$textFieldCandidates = this.inputCtrl.valueChanges.pipe(
      map((name) => {
        const nonIncluded = subtractSet(this.choices, this.selectedItems);
        if (!name) return iterToArr(nonIncluded.values());
        return filteElementsByIncludedSubstring(nonIncluded, name);
      }),
    );
  }

  removeItem(removedItem: string) {
    const ns = new Set<string>();
    this.selectedItems.forEach((a) => ns.add(a));
    ns.delete(removedItem);
    this.selectedItemsChange.emit(ns);
  }

  addItem(event: MatChipInputEvent): void {
    const value = (event.value || '').trim();

    if (value) {
      const ns = new Set<string>();
      this.selectedItems.forEach((a) => ns.add(a));
      ns.add(value);
      this.selectedItemsChange.emit(ns);
    }

    // Clear the input value
    event.chipInput!.clear();

    this.inputCtrl.setValue(null);
  }

  addAll(): void {
    this.selectedItemsChange.emit(this.choices);
  }

  removeAll(): void {
    this.selectedItemsChange.emit(new Set());
  }

  selectOnly(item: string) {
    this.selectedItemsChange.emit(new Set([item]));
  }

  selected(event: MatAutocompleteSelectedEvent): void {
    const ns = new Set<string>();
    this.selectedItems.forEach((a) => ns.add(a));
    ns.add(event.option.viewValue);
    this.selectedItemsChange.emit(ns);
    this.inputElement.nativeElement.value = '';
    this.inputCtrl.setValue(null);
  }

  onClose() {
    this.closeButtonClicked.emit();
  }
}
