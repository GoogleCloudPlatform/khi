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

import { computed, signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { Code, ConnectError } from '@connectrpc/connect';
import { of } from 'rxjs';
import { WatchIndexProgressResponse_IndexState } from 'src/app/generated/api/v1/workbench_pb';
import { MaskSmartComponent } from 'src/app/mask/mask-smart.component';
import { StringPoolIndexPhase } from 'src/app/mask/types/mask-viewmodel';
import {
  BACKEND_API,
  BackendAPI,
  DownloadProgressReporter,
} from 'src/app/services/api/backend-api-interface';
import { BackendAPIUtil } from 'src/app/services/api/backend-api.service';
import {
  MaskInspectionRuleInput,
  MaskedInspectionProgressCallback,
  StringPoolIndexProgressCallback,
  WorkbenchClientService,
} from 'src/app/services/api/workbench/workbench-client.service';
import { StringMaskService } from 'src/app/services/mask/string-mask.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';

describe('MaskSmartComponent', () => {
  let component: MaskSmartComponent;
  let fixture: ComponentFixture<MaskSmartComponent>;
  let activeWorkbenchIdSignal: WritableSignal<string | null>;
  let workbenchClientSpy: jasmine.SpyObj<WorkbenchClientService>;
  let backendAPISpy: jasmine.SpyObj<BackendAPI>;
  let progressSpy: jasmine.SpyObj<ProgressDialogStatusUpdator>;
  let stringMaskService: StringMaskService;

  beforeEach(async () => {
    activeWorkbenchIdSignal = signal<string | null>('wb-1');
    workbenchClientSpy = jasmine.createSpyObj(
      'WorkbenchClientService',
      [
        'listWorkbenchStrings',
        'createMaskedInspection',
        'watchStringPoolIndexProgress',
      ],
      {
        activeWorkbenchId: activeWorkbenchIdSignal.asReadonly(),
        isWorkbenchActive: computed(() => activeWorkbenchIdSignal() !== null),
      },
    );
    workbenchClientSpy.listWorkbenchStrings.and.returnValue(
      Promise.resolve({
        values: ['prod-secret-ns', 'web-pod-1'],
        totalCount: 2,
      }),
    );
    workbenchClientSpy.watchStringPoolIndexProgress.and.callFake(
      async (onUpdate: StringPoolIndexProgressCallback) => {
        onUpdate({
          state: WatchIndexProgressResponse_IndexState.READY,
          progressPercentage: 100,
          message: 'String pool search index ready.',
        });
      },
    );
    workbenchClientSpy.createMaskedInspection.and.returnValue(
      Promise.resolve({
        inspectionId: 'masked-1',
        inspectionName: 'demo (masked)',
        suggestedFilename: 'demo-masked.khi',
        fileSizeBytes: 1024n,
      }),
    );

    backendAPISpy = jasmine.createSpyObj('BackendAPI', ['getInspectionData']);
    progressSpy = jasmine.createSpyObj('ProgressDialogStatusUpdator', [
      'show',
      'dismiss',
      'updateProgress',
    ]);

    await TestBed.configureTestingModule({
      imports: [MaskSmartComponent, NoopAnimationsModule],
      providers: [
        StringMaskService,
        { provide: WorkbenchClientService, useValue: workbenchClientSpy },
        { provide: BACKEND_API, useValue: backendAPISpy },
        { provide: PROGRESS_DIALOG_STATUS_UPDATOR, useValue: progressSpy },
      ],
    }).compileComponents();

    stringMaskService = TestBed.inject(StringMaskService);
    fixture = TestBed.createComponent(MaskSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  });

  it('should watch string pool index progress and load initial candidates on mount', () => {
    expect(component).toBeTruthy();
    expect(
      workbenchClientSpy.watchStringPoolIndexProgress,
    ).toHaveBeenCalledTimes(1);
    expect(workbenchClientSpy.listWorkbenchStrings).toHaveBeenCalledWith(
      '',
      0,
      150,
      jasmine.any(AbortSignal),
    );
    expect(component['indexStatus']().phase).toBe(StringPoolIndexPhase.Ready);
    expect(component['candidates']().map((c) => c.value)).toEqual([
      'prod-secret-ns',
      'web-pod-1',
    ]);
  });

  it('should query with RE2 regex, surface invalid regex inline without clearing candidates, and bulk-select shown strings', async () => {
    workbenchClientSpy.listWorkbenchStrings.calls.reset();
    workbenchClientSpy.listWorkbenchStrings.and.returnValue(
      Promise.resolve({
        values: ['10.128.0.14', '10.128.0.29'],
        totalCount: 2,
      }),
    );

    await component['fetchCandidates']('^10\\.128\\..*', false);
    await fixture.whenStable();

    expect(workbenchClientSpy.listWorkbenchStrings).toHaveBeenCalledWith(
      '^10\\.128\\..*',
      0,
      150,
      jasmine.any(AbortSignal),
    );
    expect(component['candidates']().map((c) => c.value)).toEqual([
      '10.128.0.14',
      '10.128.0.29',
    ]);

    // Invalid regex syntax returns Code.InvalidArgument -> inline searchRegexError, preserving previous candidates.
    workbenchClientSpy.listWorkbenchStrings.and.returnValue(
      Promise.reject(
        new ConnectError(
          'invalid regex query: missing closing ]',
          Code.InvalidArgument,
        ),
      ),
    );
    await component['fetchCandidates']('[unclosed', false);
    expect(component['searchRegexError']()).toContain('missing closing ]');
    expect(component['errorMessage']()).toBe('');
    expect(component['candidates']().map((c) => c.value)).toEqual([
      '10.128.0.14',
      '10.128.0.29',
    ]);

    // Bulk-selecting shown candidates allocates sequential <masked_1>, <masked_2> placeholders.
    component['onSelectShownCandidates']();
    expect(
      component['rules']().map((r) => [
        r.target,
        r.replacement,
        r.isDefaultReplacement,
      ]),
    ).toEqual([
      ['10.128.0.14', '<masked_1>', true],
      ['10.128.0.29', '<masked_2>', true],
    ]);
  });

  it('should stream server masking and download progress into ProgressDialog and save the masked .khi file', async () => {
    const saveBlobSpy = spyOn(BackendAPIUtil, 'saveBlobAsFile');
    const mockBlob = new Blob(['masked-khi-bytes']);

    workbenchClientSpy.createMaskedInspection.and.callFake(
      async (
        _rules: readonly MaskInspectionRuleInput[],
        onProgress?: MaskedInspectionProgressCallback,
      ) => {
        onProgress?.(40, 'Masking string pool (400 / 1000 strings)...');
        onProgress?.(90, 'Validating masked .khi archive...');
        return {
          inspectionId: 'masked-1',
          inspectionName: 'demo (Masked)',
          suggestedFilename: 'demo-masked.khi',
          fileSizeBytes: 1024n,
        };
      },
    );
    backendAPISpy.getInspectionData.and.callFake(
      (_id: string, reporter: DownloadProgressReporter) => {
        reporter(1024, 512);
        reporter(1024, 1024);
        return of({ fileName: 'demo-masked.khi', content: mockBlob });
      },
    );

    stringMaskService.toggleTarget('web-pod-1', true);
    stringMaskService.addCustomRule('prod-', '');
    await component['onExportMaskedKhi']();

    expect(progressSpy.show).toHaveBeenCalledTimes(1);
    expect(workbenchClientSpy.createMaskedInspection).toHaveBeenCalledWith(
      [
        { target: 'web-pod-1', replacement: '<masked_1>' },
        { target: 'prod-', replacement: '' },
      ],
      jasmine.any(Function),
      jasmine.any(AbortSignal),
    );
    expect(backendAPISpy.getInspectionData).toHaveBeenCalledWith(
      'masked-1',
      jasmine.any(Function),
    );
    expect(progressSpy.updateProgress).toHaveBeenCalledWith(
      jasmine.objectContaining({
        mode: 'determinate',
        percent: 30, // 40% * 0.75
        message: 'Masking string pool (400 / 1000 strings)...',
      }),
    );
    expect(progressSpy.updateProgress).toHaveBeenCalledWith(
      jasmine.objectContaining({
        mode: 'determinate',
        percent: 100, // 75% + 100% * 0.25
      }),
    );
    expect(saveBlobSpy).toHaveBeenCalledWith(mockBlob, 'demo-masked.khi');
    expect(progressSpy.dismiss).toHaveBeenCalledTimes(1);
    expect(component['exportProgress']()).toBeNull();
    expect(component['isExporting']()).toBeFalse();
    expect(component['statusMessage']()).toContain('demo (Masked)');
  });

  it('should dismiss progress dialog and surface error message when masked export fails', async () => {
    workbenchClientSpy.createMaskedInspection.and.returnValue(
      Promise.reject(new Error('disk full')),
    );

    stringMaskService.toggleTarget('web-pod-1', true);
    await component['onExportMaskedKhi']();

    expect(progressSpy.show).toHaveBeenCalledTimes(1);
    expect(progressSpy.dismiss).toHaveBeenCalledTimes(1);
    expect(component['exportProgress']()).toBeNull();
    expect(component['isExporting']()).toBeFalse();
    expect(component['errorMessage']()).toContain('disk full');
  });
});
