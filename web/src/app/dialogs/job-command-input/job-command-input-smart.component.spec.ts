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
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { JobCommandInputLayoutComponent } from 'src/app/dialogs/job-command-input/components/job-command-input-layout.component';
import {
  JobCommandInputSmartComponent,
  openJobCommandInputDialog,
} from 'src/app/dialogs/job-command-input/job-command-input-smart.component';
import { ParsedJobCommand } from 'src/app/dialogs/startup/utils/job-command-parser';

describe('JobCommandInputSmartComponent', () => {
  let component: JobCommandInputSmartComponent;
  let fixture: ComponentFixture<JobCommandInputSmartComponent>;
  let dialogRefSpy: jasmine.SpyObj<
    MatDialogRef<JobCommandInputSmartComponent, ParsedJobCommand | null>
  >;
  let dialogSpy: jasmine.SpyObj<MatDialog>;

  beforeEach(async () => {
    dialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close']);
    dialogSpy = jasmine.createSpyObj('MatDialog', ['open']);

    await TestBed.configureTestingModule({
      imports: [JobCommandInputSmartComponent, NoopAnimationsModule],
      providers: [{ provide: MatDialogRef, useValue: dialogRefSpy }],
    }).compileComponents();

    fixture = TestBed.createComponent(JobCommandInputSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and render layout component', () => {
    expect(component).toBeTruthy();
    const layoutEl = fixture.debugElement.query(
      By.directive(JobCommandInputLayoutComponent),
    );
    expect(layoutEl).toBeTruthy();
  });

  it('should close dialog with parsed command on valid submit', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(JobCommandInputLayoutComponent),
    );
    const layout = layoutEl.componentInstance as JobCommandInputLayoutComponent;
    layout.command.set(
      './khi --job-mode --job-inspection-type="gke-audit" --job-inspection-features="f1,f2" --job-inspection-values=\'{"project":"my-p"}\'',
    );
    fixture.detectChanges();

    layoutEl.triggerEventHandler('submitCommand', undefined);

    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith({
      inspectionType: 'gke-audit',
      features: ['f1', 'f2'],
      parameters: { project: 'my-p' },
    });
  });

  it('should update errorMessage and not close dialog on invalid submit', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(JobCommandInputLayoutComponent),
    );
    const layout = layoutEl.componentInstance as JobCommandInputLayoutComponent;
    layout.command.set(
      './khi --job-mode --job-inspection-values="invalid-json"',
    );
    fixture.detectChanges();

    layoutEl.triggerEventHandler('submitCommand', undefined);
    fixture.detectChanges();

    expect(dialogRefSpy.close).not.toHaveBeenCalled();
    expect(layout.errorMessage()).toBeTruthy();
    expect(layout.errorMessage()).toContain('Missing required flag');
  });

  it('should close dialog with null on cancel', () => {
    const layoutEl = fixture.debugElement.query(
      By.directive(JobCommandInputLayoutComponent),
    );
    layoutEl.triggerEventHandler('cancelDialog', undefined);

    expect(dialogRefSpy.close).toHaveBeenCalledOnceWith(null);
  });

  describe('openJobCommandInputDialog', () => {
    it('should open dialog with configured width', () => {
      openJobCommandInputDialog(dialogSpy);
      expect(dialogSpy.open).toHaveBeenCalledWith(
        JobCommandInputSmartComponent,
        jasmine.objectContaining({
          width: '640px',
        }),
      );
    });
  });
});
