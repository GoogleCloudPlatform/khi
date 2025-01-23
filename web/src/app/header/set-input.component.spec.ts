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

import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatChipHarness } from '@angular/material/chips/testing';

import { SetInputComponent } from './set-input.component';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { HarnessLoader } from '@angular/cdk/testing';
import { MatTooltipModule } from '@angular/material/tooltip';

describe('SetInputComponent', () => {
  let component: TestSetInputWrapperComponent;
  let fixture: ComponentFixture<TestSetInputWrapperComponent>;
  let loader: HarnessLoader;

  @Component({
    template: `
      <khi-header-set-input
        label="test"
        [selectedItems]="selected"
        [choices]="choices"
        (selectedItemsChange)="onUpdate($event)"
      >
      </khi-header-set-input>
    `,
  })
  class TestSetInputWrapperComponent {
    choices = new Set(['foo', 'bar', 'qux']);

    selected = new Set(['foo', 'bar']);

    latest = new Set<string>();

    onUpdate(elements: Set<string>) {
      this.latest = elements;
    }
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [TestSetInputWrapperComponent, SetInputComponent],
      imports: [
        CommonModule,
        NoopAnimationsModule,
        MatIconModule,
        MatButtonModule,
        MatFormFieldModule,
        MatInputModule,
        MatChipsModule,
        MatAutocompleteModule,
        FormsModule,
        ReactiveFormsModule,
        MatTooltipModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TestSetInputWrapperComponent);
    component = fixture.componentInstance;
    loader = TestbedHarnessEnvironment.loader(fixture);
    fixture.detectChanges();
  });

  it('selected item should be reflected to mat-chip items', async () => {
    expect(component).toBeTruthy();

    const matChips = await loader.getAllHarnesses(MatChipHarness);
    expect(matChips.length).toBe(2);

    component.selected.delete('foo');
    fixture.detectChanges();
    const matChipsAfterDelete = await loader.getAllHarnesses(MatChipHarness);
    expect(matChipsAfterDelete.length).toBe(1);
  });

  it('Chip item should be removable by clicking cross button', async () => {
    const matChips = await loader.getAllHarnesses(MatChipHarness);
    expect(matChips.length).toBe(2);
    await matChips[0].remove();
    fixture.detectChanges();

    expect(component.latest.size).toBe(1);
  });

  it('Add all button should add every choices into selection', () => {
    expect(component).toBeTruthy();
    const setInput = fixture.debugElement.query(
      By.directive(SetInputComponent),
    );
    const setInputComponent = setInput.componentInstance as SetInputComponent;
    setInputComponent.addAll();
    expect(component.latest.size).toBe(3);
  });

  it('Remove all button should delete all', () => {
    expect(component).toBeTruthy();
    const setInput = fixture.debugElement.query(
      By.directive(SetInputComponent),
    );
    const setInputComponent = setInput.componentInstance as SetInputComponent;
    setInputComponent.removeAll();
    expect(component.latest.size).toBe(0);
  });
});
