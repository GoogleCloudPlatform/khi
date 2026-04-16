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
import { StartupDialogLayoutComponent } from './startup-dialog-layout.component';
import { Component, input, output } from '@angular/core';
import { By } from '@angular/platform-browser';
import { SidebarLink } from '../types/startup-side-menu.types';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';
import { StartupSideMenuComponent } from './startup-side-menu.component';
import { StartupDialogContentComponent } from './startup-dialog-content.component';

// Mock child components
@Component({
  selector: 'khi-startup-side-menu',
  template: '<div></div>',
})
class MockStartupSideMenuComponent {
  public readonly version = input.required<string>();
  public readonly links = input.required<SidebarLink[]>();
  public readonly newInvestigation = output<void>();
  public readonly openKhiFile = output<void>();
}

@Component({
  selector: 'khi-startup-dialog-content',
  template: '<div></div>',
})
class MockStartupDialogContentComponent {
  public readonly items = input.required<InspectionListItemViewModel[]>();
  public readonly isLoading = input<boolean>(false);
  public readonly isViewerMode = input<boolean>(false);
  public readonly openInspectionResult = output<string>();
  public readonly openInspectionMetadata = output<string>();
  public readonly cancelInspection = output<string>();
  public readonly downloadInspectionResult = output<string>();
  public readonly changeInspectionTitle = output<{
    id: string;
    changeTo: string;
  }>();
}

describe('StartupDialogLayoutComponent', () => {
  let component: StartupDialogLayoutComponent;
  let fixture: ComponentFixture<StartupDialogLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        StartupDialogLayoutComponent,
        MockStartupSideMenuComponent,
        MockStartupDialogContentComponent,
      ],
    })
      .overrideComponent(StartupDialogLayoutComponent, {
        remove: {
          imports: [StartupSideMenuComponent, StartupDialogContentComponent],
        },
        add: {
          imports: [
            MockStartupSideMenuComponent,
            MockStartupDialogContentComponent,
          ],
        },
      })
      .compileComponents();

    fixture = TestBed.createComponent(StartupDialogLayoutComponent);
    component = fixture.componentInstance;

    // Provide required inputs
    fixture.componentRef.setInput('version', '1.0.0');
    fixture.componentRef.setInput('links', []);
    fixture.componentRef.setInput('items', []);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should pass version to side menu', () => {
    fixture.componentRef.setInput('version', '2.0.0');
    fixture.detectChanges();

    const sideMenuEl = fixture.debugElement.query(
      By.directive(MockStartupSideMenuComponent),
    );
    const sideMenu =
      sideMenuEl.componentInstance as MockStartupSideMenuComponent;

    expect(sideMenu.version()).toBe('2.0.0');
  });

  it('should pass links to side menu', () => {
    const testLinks = [{ icon: 'test', label: 'Test', url: 'http://test.com' }];
    fixture.componentRef.setInput('links', testLinks);
    fixture.detectChanges();

    const sideMenuEl = fixture.debugElement.query(
      By.directive(MockStartupSideMenuComponent),
    );
    const sideMenu =
      sideMenuEl.componentInstance as MockStartupSideMenuComponent;

    expect(sideMenu.links()).toEqual(testLinks);
  });

  it('should pass items to content', () => {
    const testItems: InspectionListItemViewModel[] = [
      {
        id: '1',
        inspectionTimeLabel: '2026-04-06',
        label: 'Test',
        phase: 'DONE',
        totalProgress: {
          id: 't',
          label: 'T',
          message: 'D',
          percentage: 100,
          percentageLabel: '100%',
          indeterminate: false,
        },
        progresses: [],
        errors: [],
      },
    ];
    fixture.componentRef.setInput('items', testItems);
    fixture.detectChanges();

    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    expect(content.items()).toEqual(testItems);
  });

  it('should emit newInvestigation when side menu emits newInvestigation', () => {
    const spy = spyOn(component.newInvestigation, 'emit');
    const sideMenuEl = fixture.debugElement.query(
      By.directive(MockStartupSideMenuComponent),
    );
    const sideMenu =
      sideMenuEl.componentInstance as MockStartupSideMenuComponent;

    sideMenu.newInvestigation.emit();

    expect(spy).toHaveBeenCalled();
  });

  it('should emit openKhiFile when side menu emits openKhiFile', () => {
    const spy = spyOn(component.openKhiFile, 'emit');
    const sideMenuEl = fixture.debugElement.query(
      By.directive(MockStartupSideMenuComponent),
    );
    const sideMenu =
      sideMenuEl.componentInstance as MockStartupSideMenuComponent;

    sideMenu.openKhiFile.emit();

    expect(spy).toHaveBeenCalled();
  });

  it('should emit openInspectionResult when content emits openInspectionResult', () => {
    const spy = spyOn(component.openInspectionResult, 'emit');
    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    content.openInspectionResult.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit openInspectionMetadata when content emits openInspectionMetadata', () => {
    const spy = spyOn(component.openInspectionMetadata, 'emit');
    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    content.openInspectionMetadata.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit cancelInspection when content emits cancelInspection', () => {
    const spy = spyOn(component.cancelInspection, 'emit');
    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    content.cancelInspection.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit downloadInspectionResult when content emits downloadInspectionResult', () => {
    const spy = spyOn(component.downloadInspectionResult, 'emit');
    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    content.downloadInspectionResult.emit('test-id');

    expect(spy).toHaveBeenCalledWith('test-id');
  });

  it('should emit changeInspectionTitle when content emits changeInspectionTitle', () => {
    const spy = spyOn(component.changeInspectionTitle, 'emit');
    const contentEl = fixture.debugElement.query(
      By.directive(MockStartupDialogContentComponent),
    );
    const content =
      contentEl.componentInstance as MockStartupDialogContentComponent;

    const request = { id: 'test-id', changeTo: 'new-title' };
    content.changeInspectionTitle.emit(request);

    expect(spy).toHaveBeenCalledWith(request);
  });
});
