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
import { InspectionListItemComponent } from './inspection-list-item.component';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';
import { MatIconTestingModule } from '@angular/material/icon/testing';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { By } from '@angular/platform-browser';
import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';

describe('InspectionListItemComponent', () => {
  let component: InspectionListItemComponent;
  let fixture: ComponentFixture<InspectionListItemComponent>;

  const mockItem: InspectionListItemViewModel = {
    id: 'test-task',
    inspectionTimeLabel: 'now',
    label: 'Test Task',
    phase: 'RUNNING',
    totalProgress: {
      id: 'total',
      label: 'Progress',
      message: 'Processing',
      percentage: 50,
      percentageLabel: '50',
      indeterminate: false,
    },
    progresses: [],
    errors: [],
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        InspectionListItemComponent,
        MatIconTestingModule,
        MatButtonModule,
        MatTooltipModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionListItemComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('item', mockItem);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should switch to edit mode when title container is clicked', () => {
    const titleContainerEl = fixture.debugElement.query(
      By.css('.title-container'),
    );
    titleContainerEl.nativeElement.click();
    fixture.detectChanges();
    expect(component['isEditing']()).toBeTrue();
    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    expect(inputEl).toBeTruthy();
  });

  it('should focus the input element when entering edit mode', () => {
    const titleContainerEl = fixture.debugElement.query(
      By.css('.title-container'),
    );
    titleContainerEl.nativeElement.click();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    expect(document.activeElement).toBe(inputEl.nativeElement);
  });

  it('should switch to edit mode when enter key is pressed on title container', () => {
    const titleContainerEl = fixture.debugElement.query(
      By.css('.title-container'),
    );
    titleContainerEl.triggerEventHandler('keydown.enter', {});
    fixture.detectChanges();
    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    expect(inputEl).toBeTruthy();
  });

  it('should commit title change on blur', () => {
    component['startEditing']();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'New Title';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.nativeElement.dispatchEvent(new Event('blur'));
    fixture.detectChanges();

    const inputElAfter = fixture.debugElement.query(By.css('.title-input'));
    expect(inputElAfter).toBeNull();
    expect(component.changeInspectionTitle.emit).toHaveBeenCalledWith({
      id: 'test-task',
      changeTo: 'New Title',
    });
  });

  it('should switch to edit mode when space key is pressed on title container', () => {
    const titleContainerEl = fixture.debugElement.query(
      By.css('.title-container'),
    );
    const event = new KeyboardEvent('keydown', { key: ' ' });
    spyOn(event, 'preventDefault');
    titleContainerEl.triggerEventHandler('keydown.space', event);
    fixture.detectChanges();
    expect(component['isEditing']()).toBeTrue();
    expect(event.preventDefault).toHaveBeenCalled();
  });

  it('should commit title change on enter', () => {
    component['startEditing']();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'New Title';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.triggerEventHandler('keydown.enter', {});

    expect(component['isEditing']()).toBeFalse();
    expect(component.changeInspectionTitle.emit).toHaveBeenCalledWith({
      id: 'test-task',
      changeTo: 'New Title',
    });
  });

  it('should cancel editing on escape', () => {
    component['startEditing']();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'New Title';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.triggerEventHandler('keydown.escape', {});

    expect(component['isEditing']()).toBeFalse();
    expect(component.changeInspectionTitle.emit).not.toHaveBeenCalled();
  });

  it('should not emit changeInspectionTitle if title is unchanged', () => {
    component['startEditing']();
    fixture.detectChanges();

    const inputEl = fixture.debugElement.query(By.css('.title-input'));
    inputEl.nativeElement.value = 'Test Task';
    inputEl.nativeElement.dispatchEvent(new Event('input'));

    spyOn(component.changeInspectionTitle, 'emit');
    inputEl.nativeElement.dispatchEvent(new Event('blur'));
    fixture.detectChanges();

    expect(component.changeInspectionTitle.emit).not.toHaveBeenCalled();
  });

  it('should emit openInspectionResult when open button is clicked', () => {
    const item = { ...mockItem, phase: 'DONE' as const };
    fixture.componentRef.setInput('item', item);
    fixture.detectChanges();

    spyOn(component.openInspectionResult, 'emit');
    const button = fixture.debugElement.query(By.css('.open-button'));
    button.nativeElement.click();

    expect(component.openInspectionResult.emit).toHaveBeenCalledWith(
      'test-task',
    );
  });

  it('should emit openInspectionMetadata when metadata button is clicked', () => {
    const item = { ...mockItem, phase: 'DONE' as const };
    fixture.componentRef.setInput('item', item);
    fixture.detectChanges();

    spyOn(component.openInspectionMetadata, 'emit');
    const button = fixture.debugElement.query(By.css('.metadata-button'));
    button.nativeElement.click();

    expect(component.openInspectionMetadata.emit).toHaveBeenCalledWith(
      'test-task',
    );
  });

  it('should emit cancelInspection when cancel button is clicked', () => {
    const item = { ...mockItem, phase: 'RUNNING' as const };
    fixture.componentRef.setInput('item', item);
    fixture.detectChanges();

    spyOn(component.cancelInspection, 'emit');
    const button = fixture.debugElement.query(By.css('.cancel-button'));
    button.nativeElement.click();

    expect(component.cancelInspection.emit).toHaveBeenCalledWith('test-task');
  });

  it('should emit downloadInspectionResult when download button is clicked', () => {
    const item = { ...mockItem, phase: 'DONE' as const };
    fixture.componentRef.setInput('item', item);
    fixture.detectChanges();

    spyOn(component.downloadInspectionResult, 'emit');
    const button = fixture.debugElement.query(By.css('.download-button'));
    button.nativeElement.click();

    expect(component.downloadInspectionResult.emit).toHaveBeenCalledWith(
      'test-task',
    );
  });

  function checkButtonVisibilityForPhase(
    phase: InspectionMetadataProgressPhase,
    buttonClass: string,
    wantVisible: boolean,
  ) {
    it(`should ${wantVisible ? 'show' : 'not show'} ${buttonClass} on phase=${phase}`, () => {
      const item2 = {
        ...mockItem,
      };
      item2.phase = phase;
      fixture.componentRef.setInput('item', item2);
      fixture.detectChanges();
      const openBtn = fixture.debugElement.query(By.css(buttonClass));
      if (wantVisible) {
        expect(openBtn).not.toBeNull();
      } else {
        expect(openBtn).toBeNull();
      }
    });
  }

  checkButtonVisibilityForPhase('RUNNING', '.open-button', false);
  checkButtonVisibilityForPhase('DONE', '.open-button', true);
  checkButtonVisibilityForPhase('ERROR', '.open-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.open-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.metadata-button', false);
  checkButtonVisibilityForPhase('DONE', '.metadata-button', true);
  checkButtonVisibilityForPhase('ERROR', '.metadata-button', true);
  checkButtonVisibilityForPhase('CANCELLED', '.metadata-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.cancel-button', true);
  checkButtonVisibilityForPhase('DONE', '.cancel-button', false);
  checkButtonVisibilityForPhase('ERROR', '.cancel-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.cancel-button', false);

  checkButtonVisibilityForPhase('RUNNING', '.download-button', false);
  checkButtonVisibilityForPhase('DONE', '.download-button', true);
  checkButtonVisibilityForPhase('ERROR', '.download-button', false);
  checkButtonVisibilityForPhase('CANCELLED', '.download-button', false);
});
