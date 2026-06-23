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
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';

describe('ReleaseNotesDialogSmartComponent', () => {
  let component: ReleaseNotesDialogSmartComponent;
  let fixture: ComponentFixture<ReleaseNotesDialogSmartComponent>;
  let dialogRefSpy: jasmine.SpyObj<
    MatDialogRef<ReleaseNotesDialogSmartComponent>
  >;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    dialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close']);
    localStorage.removeItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY);

    await TestBed.configureTestingModule({
      imports: [
        ReleaseNotesDialogSmartComponent,
        MatDialogModule,
        NoopAnimationsModule,
      ],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: MatDialogRef,
          useValue: dialogRefSpy,
        },
      ],
    }).compileComponents();

    httpMock = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ReleaseNotesDialogSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();

    const req = httpMock.expectOne('assets/release_note/release_note.md');
    req.flush('# Mock Release Notes');
    fixture.detectChanges();
  });

  afterEach(() => {
    httpMock.verify();
    localStorage.removeItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY);
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

  it('should close dialog without saving to localStorage when doNotShowAgain is false', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(ReleaseNotesLayoutComponent),
    );
    layoutEl.triggerEventHandler('closed', undefined);

    expect(
      localStorage.getItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY),
    ).toBeNull();
    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith();
  });

  it('should save version to localStorage when doNotShowAgain is true on close', () => {
    fixture.debugElement
      .query(By.directive(ReleaseNotesLayoutComponent))
      .componentInstance.doNotShowAgain.set(true);

    const layoutEl = fixture.debugElement.query(
      By.directive(ReleaseNotesLayoutComponent),
    );
    layoutEl.triggerEventHandler('closed', undefined);

    expect(localStorage.getItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY)).toBe(
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
      openReleaseNotesDialog(dialogSpy);
      expect(dialogSpy.open).toHaveBeenCalledOnceWith(
        ReleaseNotesDialogSmartComponent,
        jasmine.any(Object),
      );
    });

    it('should not open dialog when suppressed for current version', () => {
      localStorage.setItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY, VERSION);
      const result = openReleaseNotesDialog(dialogSpy);
      expect(result).toBeNull();
      expect(dialogSpy.open).not.toHaveBeenCalled();
    });

    it('should open dialog when suppressed if force is true', () => {
      localStorage.setItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY, VERSION);
      openReleaseNotesDialog(dialogSpy, true);
      expect(dialogSpy.open).toHaveBeenCalledOnceWith(
        ReleaseNotesDialogSmartComponent,
        jasmine.any(Object),
      );
    });
  });
});
