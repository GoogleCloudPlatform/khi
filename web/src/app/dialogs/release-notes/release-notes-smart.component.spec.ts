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
import {
  MatDialog,
  MatDialogModule,
  MatDialogRef,
} from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import {
  openReleaseNotesDialog,
  ReleaseNotesDialogSmartComponent,
  SUPPRESSED_RELEASE_NOTES_VERSION_KEY,
} from 'src/app/dialogs/release-notes/release-notes-smart.component';
import { VERSION } from 'src/environments/version';
import { By } from '@angular/platform-browser';
import { ReleaseNotesLayoutComponent } from 'src/app/dialogs/release-notes/components/release-notes-layout.component';
import {
  SETTINGS_STORAGE,
  SettingsStorage,
} from 'src/app/services/settings/settings-storage';

describe('ReleaseNotesDialogSmartComponent', () => {
  let component: ReleaseNotesDialogSmartComponent;
  let fixture: ComponentFixture<ReleaseNotesDialogSmartComponent>;
  let dialogRefSpy: jasmine.SpyObj<
    MatDialogRef<ReleaseNotesDialogSmartComponent>
  >;
  let settingsStorageSpy: jasmine.SpyObj<SettingsStorage>;

  beforeEach(async () => {
    dialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close']);
    settingsStorageSpy = jasmine.createSpyObj('SettingsStorage', [
      'getItem',
      'setItem',
    ]);

    spyOn(window, 'fetch').and.resolveTo(
      new Response('# Mock Release Notes', { status: 200 }),
    );

    await TestBed.configureTestingModule({
      imports: [
        ReleaseNotesDialogSmartComponent,
        MatDialogModule,
        NoopAnimationsModule,
      ],
      providers: [
        {
          provide: MatDialogRef,
          useValue: dialogRefSpy,
        },
        {
          provide: SETTINGS_STORAGE,
          useValue: settingsStorageSpy,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ReleaseNotesDialogSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  });

  it('should create and fetch release notes from assets', () => {
    expect(component).toBeTruthy();
    const layoutEl = fixture.debugElement.query(
      By.directive(ReleaseNotesLayoutComponent),
    );
    expect(layoutEl.componentInstance.markdownContent()).toBe(
      '# Mock Release Notes',
    );
  });

  it('should close dialog without saving to settingsStorage when doNotShowAgain is false', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(ReleaseNotesLayoutComponent),
    );
    layoutEl.triggerEventHandler('closed', undefined);

    expect(settingsStorageSpy.setItem).not.toHaveBeenCalled();
    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith();
  });

  it('should save version to settingsStorage when doNotShowAgain is true on close', () => {
    fixture.debugElement
      .query(By.directive(ReleaseNotesLayoutComponent))
      .componentInstance.doNotShowAgain.set(true);

    const layoutEl = fixture.debugElement.query(
      By.directive(ReleaseNotesLayoutComponent),
    );
    layoutEl.triggerEventHandler('closed', undefined);

    expect(settingsStorageSpy.setItem).toHaveBeenCalledOnceWith(
      SUPPRESSED_RELEASE_NOTES_VERSION_KEY,
      VERSION,
    );
    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith();
  });

  describe('openReleaseNotesDialog', () => {
    let dialogSpy: jasmine.SpyObj<MatDialog>;

    beforeEach(() => {
      dialogSpy = jasmine.createSpyObj('MatDialog', ['open']);
    });

    it('should open dialog when not suppressed', () => {
      settingsStorageSpy.getItem.and.returnValue(null);
      openReleaseNotesDialog(dialogSpy, settingsStorageSpy);
      expect(dialogSpy.open).toHaveBeenCalledOnceWith(
        ReleaseNotesDialogSmartComponent,
        jasmine.any(Object),
      );
    });

    it('should not open dialog when suppressed for current version', () => {
      settingsStorageSpy.getItem.and.returnValue(VERSION);
      const result = openReleaseNotesDialog(dialogSpy, settingsStorageSpy);
      expect(result).toBeNull();
      expect(dialogSpy.open).not.toHaveBeenCalled();
    });

    it('should open dialog when suppressed if force is true', () => {
      settingsStorageSpy.getItem.and.returnValue(VERSION);
      openReleaseNotesDialog(dialogSpy, settingsStorageSpy, true);
      expect(dialogSpy.open).toHaveBeenCalledOnceWith(
        ReleaseNotesDialogSmartComponent,
        jasmine.any(Object),
      );
    });
  });
});
