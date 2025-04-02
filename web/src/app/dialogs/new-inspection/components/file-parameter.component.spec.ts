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
import { FileParameterComponent } from './file-parameter.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from '@angular/platform-browser-dynamic/testing';
import { MatIconModule, MatIconRegistry } from '@angular/material/icon';
import { FILE_UPLOADER, MockFileUploader } from './service/file-uploader';
import { By } from '@angular/platform-browser';
import { of } from 'rxjs';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { MatProgressSpinnerHarness } from '@angular/material/progress-spinner/testing';
import {
  FileParameterFormField,
  ParameterHintType,
  ParameterInputType,
  UploadStatus,
  UploadToken,
} from '../../../common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';

describe('FileParameterComponent', () => {
  const mockFileUploader = new MockFileUploader();
  const fakeUploadToken: UploadToken = { id: 'foo' };
  const defaultFileParameterForm = {
    label: 'test-field-label',
    description: 'test-description',
    hintType: ParameterHintType.Info,
    hint: 'this is a test info',
    type: ParameterInputType.File,
    id: 'test-id',
    token: fakeUploadToken,
    status: UploadStatus.Waiting,
  } as FileParameterFormField;

  let fixture: ComponentFixture<FileParameterComponent>;
  beforeAll(() => {
    TestBed.resetTestEnvironment();
    TestBed.initTestEnvironment(
      BrowserDynamicTestingModule,
      platformBrowserDynamicTesting(),
      { teardown: { destroyAfterEach: false } },
    );
  });

  beforeEach(async () => {
    mockFileUploader.statusProvider =
      MockFileUploader.MOCK_COMPLETED_UPLOAD_STATUS_PROVIDER;
    await TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule, MatIconModule],
      providers: [
        {
          provide: FILE_UPLOADER,
          useValue: mockFileUploader,
        },
        {
          provide: PARAMETER_STORE,
          useClass: DefaultParameterStore,
        },
      ],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(FileParameterComponent);
    fixture.componentRef.setInput('parameter', defaultFileParameterForm);
  });

  it('should pass input values', () => {
    fixture.detectChanges();

    expect(fixture.componentInstance).toBeTruthy();
  });

  it('shows filename if the name is assigned', () => {
    fixture.componentInstance.processReceivedFileInfo([
      new File([], 'test-filename.txt'),
    ]);
    fixture.detectChanges();

    const dropAreaFilename = fixture.debugElement.query(
      By.css('.drop-area-hint-file-name > span'),
    );
    expect(dropAreaFilename.nativeElement.textContent).toBe(
      'test-filename.txt',
    );
    const uploadButton = fixture.debugElement.query(By.css('.upload-button'));
    expect(uploadButton.attributes['disabled']).toBeFalsy();
  });

  it('shows progress bar with upload status', async () => {
    mockFileUploader.statusProvider = () =>
      of({
        done: false,
        completeRatio: 0.5,
        completeRatioUnknown: false,
      });

    fixture.componentRef.setInput('parameter', {
      ...defaultFileParameterForm,
      status: UploadStatus.Uploading,
    });
    fixture.componentInstance.selectedFile = new File([], 'a mock file');
    fixture.componentInstance.onClickUploadButton();
    fixture.detectChanges();
    const harnessLoader = TestbedHarnessEnvironment.loader(fixture);
    const spinner = await harnessLoader.getHarness(MatProgressSpinnerHarness);

    expect(fixture.componentInstance).toBeTruthy();
    expect(await spinner.getMode()).toBe('determinate');
    expect(await spinner.getValue()).toBe(50);
  });

  it('shows progress bar with veryfying status', async () => {
    mockFileUploader.statusProvider = () =>
      of({
        done: true,
        completeRatio: 1,
        completeRatioUnknown: false,
      });

    fixture.componentRef.setInput('parameter', {
      ...defaultFileParameterForm,
      status: UploadStatus.Verifying,
    });
    fixture.componentInstance.selectedFile = new File([], 'a mock file');
    fixture.componentInstance.onClickUploadButton();
    fixture.detectChanges();
    const harnessLoader = TestbedHarnessEnvironment.loader(fixture);
    const spinner = await harnessLoader.getHarness(MatProgressSpinnerHarness);

    expect(fixture.componentInstance).toBeTruthy();
    expect(await spinner.getMode()).toBe('indeterminate');
  });

  it('shows done message with done status', async () => {
    mockFileUploader.statusProvider = () =>
      of({
        done: false,
        completeRatio: 0.5,
        completeRatioUnknown: false,
      });
    fixture.componentRef.setInput('parameter', {
      ...defaultFileParameterForm,
      status: UploadStatus.Done,
    });
    fixture.componentInstance.onClickUploadButton();
    fixture.detectChanges();

    const doneLabel = fixture.debugElement.query(
      By.css('.done-status-indicator-label'),
    );
    expect(doneLabel).not.toBeNull();
  });

  it('must disable upload button after upload', () => {
    fixture.componentRef.setInput('parameter', {
      ...defaultFileParameterForm,
      status: UploadStatus.Verifying,
    });
    fixture.componentInstance.selectedFile = new File([], 'a mock file');
    fixture.componentInstance.isSelectedFileUploaded.set(false);
    fixture.detectChanges();
    const uploadButton = fixture.debugElement.query(By.css('.upload-button'));
    expect(uploadButton.attributes['disabled']).toBeFalsy();

    fixture.componentInstance.onClickUploadButton();
    fixture.detectChanges();

    expect(uploadButton.attributes['disabled']).toBeTruthy();
  });
});
