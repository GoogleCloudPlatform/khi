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
import { StyleOverrideLayoutComponent } from 'src/app/dialogs/style-override/components/style-override-layout.component';
import { By } from '@angular/platform-browser';
import { MatDialogModule } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RevisionStateStyle } from 'src/app/store/domain/style';
import { RevisionStateOverrideListComponent } from 'src/app/dialogs/style-override/components/revision-state-override-list.component';
import { TimelineTypeOverrideListComponent } from 'src/app/dialogs/style-override/components/timeline-type-override-list.component';
import { LogTypeOverrideListComponent } from 'src/app/dialogs/style-override/components/log-type-override-list.component';
import {
  TimelineTypeOverrideEvent,
  LogTypeOverrideEvent,
} from 'src/app/dialogs/style-override/types/style-override-viewmodel';

describe('StyleOverrideLayoutComponent', () => {
  let component: StyleOverrideLayoutComponent;
  let fixture: ComponentFixture<StyleOverrideLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        StyleOverrideLayoutComponent,
        MatDialogModule,
        NoopAnimationsModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(StyleOverrideLayoutComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.componentRef.setInput('revisionStates', []);
    fixture.componentRef.setInput('timelineTypes', []);
    fixture.componentRef.setInput('logTypes', []);
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should render mat-tab-group and pass inputs down', () => {
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
    fixture.componentRef.setInput('timelineTypes', [
      {
        id: 1,
        label: 'Pod',
        icon: 'pod',
        description: 'Pod type',
        hexColor: '#3f51b5',
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.componentRef.setInput('logTypes', [
      {
        id: 1,
        label: 'Audit',
        description: 'Audit log events',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    const tabs = fixture.debugElement.queryAll(By.css('.mat-mdc-tab'));
    expect(tabs.length).toBe(3);

    const revisionList = fixture.debugElement.query(
      By.directive(RevisionStateOverrideListComponent),
    );
    expect(revisionList).toBeTruthy();
    expect(revisionList.componentInstance.revisionStates()).toEqual(
      component.revisionStates(),
    );

    // Select and detect changes for Timeline Types tab
    const tabGroup = fixture.debugElement.query(
      By.css('mat-tab-group'),
    ).componentInstance;
    tabGroup.selectedIndex = 1;
    fixture.detectChanges();

    const timelineTypeList = fixture.debugElement.query(
      By.directive(TimelineTypeOverrideListComponent),
    );
    expect(timelineTypeList).toBeTruthy();
    expect(timelineTypeList.componentInstance.timelineTypes()).toEqual(
      component.timelineTypes(),
    );

    // Select and detect changes for Log Types tab
    tabGroup.selectedIndex = 2;
    fixture.detectChanges();

    const logTypeList = fixture.debugElement.query(
      By.directive(LogTypeOverrideListComponent),
    );
    expect(logTypeList).toBeTruthy();
    expect(logTypeList.componentInstance.logTypes()).toEqual(
      component.logTypes(),
    );
  });

  it('should forward revisionStateColorChange and revisionStateResetColor', () => {
    fixture.componentRef.setInput('revisionStates', []);
    fixture.componentRef.setInput('timelineTypes', []);
    fixture.componentRef.setInput('logTypes', []);
    fixture.detectChanges();

    let colorChangeEmitted: { id: number; hexColor: string } | undefined;
    component.revisionStateColorChange.subscribe(
      (event) => (colorChangeEmitted = event),
    );

    let resetColorEmitted: number | undefined;
    component.revisionStateResetColor.subscribe(
      (id) => (resetColorEmitted = id),
    );

    const revisionList = fixture.debugElement.query(
      By.directive(RevisionStateOverrideListComponent),
    ).componentInstance as RevisionStateOverrideListComponent;

    revisionList.colorChange.emit({ id: 2, hexColor: '#ffffff' });
    expect(colorChangeEmitted).toEqual({ id: 2, hexColor: '#ffffff' });

    revisionList.resetColor.emit(3);
    expect(resetColorEmitted).toBe(3);
  });

  it('should forward timelineTypePropertyChange and timelineTypeResetColor', () => {
    fixture.componentRef.setInput('revisionStates', []);
    fixture.componentRef.setInput('timelineTypes', []);
    fixture.componentRef.setInput('logTypes', []);
    fixture.detectChanges();

    const tabGroup = fixture.debugElement.query(
      By.css('mat-tab-group'),
    ).componentInstance;
    tabGroup.selectedIndex = 1;
    fixture.detectChanges();

    let propertyChangeEmitted: TimelineTypeOverrideEvent | undefined;
    component.timelineTypePropertyChange.subscribe(
      (event) => (propertyChangeEmitted = event),
    );

    let resetColorEmitted: number | undefined;
    component.timelineTypeResetColor.subscribe(
      (id) => (resetColorEmitted = id),
    );

    const timelineList = fixture.debugElement.query(
      By.directive(TimelineTypeOverrideListComponent),
    ).componentInstance as TimelineTypeOverrideListComponent;

    timelineList.propertyChange.emit({ id: 5, backgroundColor: '#000000' });
    expect(propertyChangeEmitted).toEqual({
      id: 5,
      backgroundColor: '#000000',
    });

    timelineList.resetColor.emit(6);
    expect(resetColorEmitted).toBe(6);
  });

  it('should forward logTypePropertyChange and logTypeResetColor', () => {
    fixture.componentRef.setInput('revisionStates', []);
    fixture.componentRef.setInput('timelineTypes', []);
    fixture.componentRef.setInput('logTypes', []);
    fixture.detectChanges();

    const tabGroup = fixture.debugElement.query(
      By.css('mat-tab-group'),
    ).componentInstance;
    tabGroup.selectedIndex = 2;
    fixture.detectChanges();

    let propertyChangeEmitted: LogTypeOverrideEvent | undefined;
    component.logTypePropertyChange.subscribe(
      (event) => (propertyChangeEmitted = event),
    );

    let resetColorEmitted: number | undefined;
    component.logTypeResetColor.subscribe((id) => (resetColorEmitted = id));

    const logList = fixture.debugElement.query(
      By.directive(LogTypeOverrideListComponent),
    ).componentInstance as LogTypeOverrideListComponent;

    logList.propertyChange.emit({ id: 5, backgroundColor: '#000000' });
    expect(propertyChangeEmitted).toEqual({
      id: 5,
      backgroundColor: '#000000',
    });

    logList.resetColor.emit(6);
    expect(resetColorEmitted).toBe(6);
  });
});
