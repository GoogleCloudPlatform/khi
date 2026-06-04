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
import { RevisionStateOverrideListComponent } from './revision-state-override-list.component';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RevisionStateStyle } from 'src/app/store/domain/style';

describe('RevisionStateOverrideListComponent', () => {
  let component: RevisionStateOverrideListComponent;
  let fixture: ComponentFixture<RevisionStateOverrideListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RevisionStateOverrideListComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(RevisionStateOverrideListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.componentRef.setInput('revisionStates', []);
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should list states', () => {
    fixture.componentRef.setInput('revisionStates', [
      {
        id: 1,
        label: 'Running',
        icon: 'play_arrow',
        description: 'Resource is running',
        hexColor: '#00ff00',
        isOverridden: false,
        style: RevisionStateStyle.NORMAL,
        goColorCode: 'style.Color{R: 0.000, G: 1.000, B: 0.000, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    const labelEl = fixture.debugElement.query(By.css('.state-label'));
    expect(labelEl.nativeElement.innerText).toBe('Running');
  });

  it('should emit colorChange on input', () => {
    fixture.componentRef.setInput('revisionStates', [
      {
        id: 1,
        label: 'Running',
        icon: 'play_arrow',
        description: 'Resource is running',
        hexColor: '#00ff00',
        isOverridden: false,
        style: RevisionStateStyle.NORMAL,
        goColorCode: 'style.Color{R: 0.000, G: 1.000, B: 0.000, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    let emitted: { id: number; hexColor: string } | undefined;
    component.colorChange.subscribe((event) => (emitted = event));

    const colorInput = fixture.debugElement.query(
      By.css('.color-picker-input'),
    );
    colorInput.nativeElement.value = '#0000ff';
    colorInput.nativeElement.dispatchEvent(new Event('input'));

    expect(emitted).toEqual({ id: 1, hexColor: '#0000ff' });
  });
});
