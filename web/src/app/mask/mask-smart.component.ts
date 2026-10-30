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

import {
  Component,
  OnDestroy,
  computed,
  effect,
  inject,
  signal,
} from '@angular/core';
import { Code, ConnectError } from '@connectrpc/connect';
import { firstValueFrom } from 'rxjs';
import { WatchIndexProgressResponse_IndexState } from 'src/app/generated/api/v1/workbench_pb';
import { MaskLayoutComponent } from 'src/app/mask/components/mask-layout.component';
import {
  CandidateToggleEvent,
  MaskCandidateItemViewModel,
  MaskExportPhase,
  MaskExportProgressViewModel,
  MaskRuleItemViewModel,
  RuleEnabledChangeEvent,
  RuleReplacementChangeEvent,
  StringPoolIndexPhase,
  StringPoolIndexStatusViewModel,
} from 'src/app/mask/types/mask-viewmodel';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { BackendAPIUtil } from 'src/app/services/api/backend-api.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { StringMaskService } from 'src/app/services/mask/string-mask.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import { ProgressUtil } from 'src/app/services/progress/progress-util';

/**
 * Smart container component for the dockable String Masking and Masked .khi Export pane.
 */
@Component({
  selector: 'khi-mask-smart',
  imports: [MaskLayoutComponent],
  templateUrl: './mask-smart.component.html',
  styleUrls: ['./mask-smart.component.scss'],
})
export class MaskSmartComponent implements OnDestroy {
  private static readonly PAGE_SIZE = 150;
  private static readonly SEARCH_DEBOUNCE_MS = 200;

  private readonly workbenchClient = inject(WorkbenchClientService);
  private readonly stringMaskService = inject(StringMaskService);
  private readonly backendAPI = inject<BackendAPI>(BACKEND_API);
  private readonly progress = inject<ProgressDialogStatusUpdator>(
    PROGRESS_DIALOG_STATUS_UPDATOR,
  );

  private readonly searchQuerySignal = signal<string>('');
  private readonly searchRegexErrorSignal = signal<string>('');
  private readonly rawCandidatesSignal = signal<readonly string[]>([]);
  private readonly totalCandidateCountSignal = signal<number>(0);
  private readonly isLoadingCandidatesSignal = signal<boolean>(false);
  private readonly isExportingSignal = signal<boolean>(false);
  private readonly exportProgressSignal =
    signal<MaskExportProgressViewModel | null>(null);
  private readonly statusMessageSignal = signal<string>('');
  private readonly errorMessageSignal = signal<string>('');
  private readonly indexStatusSignal = signal<StringPoolIndexStatusViewModel>({
    phase: StringPoolIndexPhase.Idle,
    progressPercentage: 0,
    message: '',
  });

  private indexWatchAbortController: AbortController | null = null;
  private listStringsAbortController: AbortController | null = null;
  private exportAbortController: AbortController | null = null;
  private searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

  /**
   * Whether an active Workbench inspection session is currently open.
   */
  protected readonly isWorkbenchActive = this.workbenchClient.isWorkbenchActive;

  /**
   * Current construction status of the on-demand string pool trigram index.
   */
  protected readonly indexStatus = this.indexStatusSignal.asReadonly();

  /**
   * Whether live masking preview across KHI views is enabled.
   */
  protected readonly livePreviewEnabled =
    this.stringMaskService.livePreviewEnabled;

  /**
   * Active search query filtering strings in the string pool.
   */
  protected readonly searchQuery = this.searchQuerySignal.asReadonly();

  /**
   * Inline RE2 syntax validation error returned by the backend for the active regex query.
   */
  protected readonly searchRegexError =
    this.searchRegexErrorSignal.asReadonly();

  /**
   * Total count of unique strings matching the search query.
   */
  protected readonly totalCandidateCount =
    this.totalCandidateCountSignal.asReadonly();

  /**
   * Whether candidate strings are currently being loaded.
   */
  protected readonly isLoadingCandidates =
    this.isLoadingCandidatesSignal.asReadonly();

  /**
   * Whether masked .khi generation and download is in progress.
   */
  protected readonly isExporting = this.isExportingSignal.asReadonly();

  /**
   * Live determinate progress details while generating and downloading a masked .khi archive.
   */
  protected readonly exportProgress = this.exportProgressSignal.asReadonly();

  /**
   * Status feedback message displayed in the Mask pane footer.
   */
  protected readonly statusMessage = this.statusMessageSignal.asReadonly();

  /**
   * Error feedback message displayed in the Mask pane footer.
   */
  protected readonly errorMessage = this.errorMessageSignal.asReadonly();

  /**
   * Candidate string items paired with their current selection state.
   */
  protected readonly candidates = computed<
    readonly MaskCandidateItemViewModel[]
  >(() => {
    const enabledSet = this.stringMaskService.enabledTargets();
    const ruleSet = this.stringMaskService.ruleTargets();
    return this.rawCandidatesSignal().map((value) => ({
      value,
      selected: enabledSet.has(value),
      inRuleList: ruleSet.has(value),
    }));
  });

  /**
   * Configured masking rules paired with deletion and default flags.
   */
  protected readonly rules = computed<readonly MaskRuleItemViewModel[]>(() =>
    this.stringMaskService.rules().map((rule) => ({
      target: rule.target,
      replacement: rule.replacement,
      defaultReplacement: rule.defaultReplacement,
      isDeletion: rule.replacement === '',
      isDefaultReplacement: rule.replacement === rule.defaultReplacement,
      enabled: rule.enabled,
    })),
  );

  /**
   * Count of enabled rules ready for export.
   */
  protected readonly enabledRuleCount = computed<number>(
    () => this.stringMaskService.enabledRules().length,
  );

  constructor() {
    effect(() => {
      const workbenchId = this.workbenchClient.activeWorkbenchId();
      this.cancelInFlightOperations();
      this.rawCandidatesSignal.set([]);
      this.totalCandidateCountSignal.set(0);
      this.searchRegexErrorSignal.set('');
      this.statusMessageSignal.set('');
      this.errorMessageSignal.set('');

      if (!workbenchId) {
        this.indexStatusSignal.set({
          phase: StringPoolIndexPhase.Idle,
          progressPercentage: 0,
          message: '',
        });
        return;
      }

      this.startStringPoolIndexWatch();
      void this.fetchCandidates(this.searchQuerySignal(), false);
    });
  }

  /**
   * Cancels active streaming watches and debounce timers on destroy.
   */
  public ngOnDestroy(): void {
    this.cancelInFlightOperations();
  }

  protected onLivePreviewChange(enabled: boolean): void {
    this.stringMaskService.setLivePreviewEnabled(enabled);
  }

  protected onSearchQueryChange(query: string): void {
    this.searchQuerySignal.set(query);
    if (this.searchDebounceTimer !== null) {
      clearTimeout(this.searchDebounceTimer);
      this.searchDebounceTimer = null;
    }
    if (query === '') {
      this.searchRegexErrorSignal.set('');
    }
    if (!this.workbenchClient.isWorkbenchActive()) {
      return;
    }
    if (query === '') {
      void this.fetchCandidates('', false);
      return;
    }
    this.searchDebounceTimer = setTimeout(() => {
      this.searchDebounceTimer = null;
      void this.fetchCandidates(query, false);
    }, MaskSmartComponent.SEARCH_DEBOUNCE_MS);
  }

  protected onSelectShownCandidates(): void {
    this.stringMaskService.addTargets(this.rawCandidatesSignal());
  }

  protected onLoadMoreCandidates(): void {
    if (this.isLoadingCandidatesSignal()) {
      return;
    }
    void this.fetchCandidates(this.searchQuerySignal(), true);
  }

  protected onCandidateToggle(event: CandidateToggleEvent): void {
    this.stringMaskService.toggleTarget(event.target, event.checked);
  }

  protected onAddCustomSubstring(target: string): void {
    this.stringMaskService.addCustomRule(target);
  }

  protected onRuleEnabledChange(event: RuleEnabledChangeEvent): void {
    this.stringMaskService.toggleTarget(event.target, event.enabled);
  }

  protected onRuleReplacementChange(event: RuleReplacementChangeEvent): void {
    this.stringMaskService.updateReplacement(event.target, event.replacement);
  }

  protected onRuleResetReplacement(target: string): void {
    this.stringMaskService.resetReplacement(target);
  }

  protected onRuleRemove(target: string): void {
    this.stringMaskService.removeRule(target);
  }

  protected onClearAllRules(): void {
    this.stringMaskService.clearAllRules();
  }

  protected async onExportMaskedKhi(): Promise<void> {
    const enabledRules = this.stringMaskService.enabledRules();
    if (enabledRules.length === 0 || this.isExportingSignal()) {
      return;
    }

    const abortCtrl = new AbortController();
    this.exportAbortController = abortCtrl;
    this.isExportingSignal.set(true);
    this.errorMessageSignal.set('');
    this.statusMessageSignal.set('');

    this.progress.show();
    this.reportExportProgress(
      MaskExportPhase.GeneratingArchive,
      0,
      'Starting (0%)',
      0,
      'Waiting for archive...',
      'Preparing masked .khi archive on server...',
    );

    try {
      const created = await this.workbenchClient.createMaskedInspection(
        enabledRules.map((rule) => ({
          target: rule.target,
          replacement: rule.replacement,
        })),
        (pct, message) => {
          if (abortCtrl.signal.aborted) {
            return;
          }
          const detail = message || 'Masking dataset strings...';
          this.reportExportProgress(
            MaskExportPhase.GeneratingArchive,
            pct,
            `${Math.round(pct)}%`,
            0,
            'Waiting for archive...',
            detail,
          );
        },
        abortCtrl.signal,
      );

      if (abortCtrl.signal.aborted) {
        return;
      }

      const targetLabel = created.suggestedFilename || created.inspectionName;
      this.reportExportProgress(
        MaskExportPhase.DownloadingFile,
        100,
        'Complete',
        0,
        'Starting download...',
        `Downloading "${targetLabel}"...`,
      );

      const downloaded = await firstValueFrom(
        this.backendAPI.getInspectionData(
          created.inspectionId,
          (fileSize, doneBytes) => {
            if (abortCtrl.signal.aborted) {
              return;
            }
            const dlPercent = fileSize > 0 ? (doneBytes / fileSize) * 100 : 100;
            const byteSummary =
              fileSize > 0
                ? ProgressUtil.formatPogressMessageByBytes(doneBytes, fileSize)
                : 'Complete';
            this.reportExportProgress(
              MaskExportPhase.DownloadingFile,
              100,
              'Complete',
              dlPercent,
              `${Math.round(dlPercent)}%`,
              `Downloading masked .khi (${byteSummary})`,
            );
          },
        ),
      );

      if (abortCtrl.signal.aborted) {
        return;
      }

      const downloadFilename = created.suggestedFilename || downloaded.fileName;
      BackendAPIUtil.saveBlobAsFile(downloaded.content, downloadFilename);
      this.statusMessageSignal.set(
        `Exported "${created.inspectionName}" and started download.`,
      );
    } catch (err) {
      if (!abortCtrl.signal.aborted) {
        const detail = err instanceof Error ? err.message : String(err);
        this.errorMessageSignal.set(`Failed to export masked .khi: ${detail}`);
      }
    } finally {
      this.progress.dismiss();
      if (this.exportAbortController === abortCtrl) {
        this.exportAbortController = null;
      }
      this.exportProgressSignal.set(null);
      this.isExportingSignal.set(false);
    }
  }

  private reportExportProgress(
    phase: MaskExportPhase,
    maskPercent: number,
    maskMessage: string,
    downloadPercent: number,
    downloadMessage: string,
    detailMessage: string,
  ): void {
    const clampedMask = Math.min(100, Math.max(0, maskPercent));
    const clampedDownload = Math.min(100, Math.max(0, downloadPercent));
    const overallPercent = clampedMask * 0.75 + clampedDownload * 0.25;
    const phaseLabel =
      phase === MaskExportPhase.GeneratingArchive
        ? 'Generating masked .khi archive...'
        : 'Downloading masked .khi file...';

    this.exportProgressSignal.set({
      phase,
      phaseLabel,
      progressPercentage: overallPercent,
      message: detailMessage,
    });

    this.progress.updateProgress({
      mode: 'determinate',
      percent: overallPercent,
      message: detailMessage,
      tasks: [
        {
          id: 'mask-archive',
          label: 'Mask & Archive',
          message: maskMessage,
          percent: clampedMask,
          mode: 'determinate',
        },
        {
          id: 'download-khi',
          label: 'Download .khi',
          message: downloadMessage,
          percent: clampedDownload,
          mode: 'determinate',
        },
      ],
    });
  }

  private startStringPoolIndexWatch(): void {
    const abortCtrl = new AbortController();
    this.indexWatchAbortController = abortCtrl;
    this.indexStatusSignal.set({
      phase: StringPoolIndexPhase.Building,
      progressPercentage: 0,
      message: 'Initializing string pool search index...',
    });

    void this.workbenchClient.watchStringPoolIndexProgress((update) => {
      if (abortCtrl.signal.aborted) {
        return;
      }
      let phase = StringPoolIndexPhase.Building;
      if (update.state === WatchIndexProgressResponse_IndexState.READY) {
        phase = StringPoolIndexPhase.Ready;
      } else if (
        update.state === WatchIndexProgressResponse_IndexState.FAILED
      ) {
        phase = StringPoolIndexPhase.Failed;
      }
      this.indexStatusSignal.set({
        phase,
        progressPercentage: update.progressPercentage,
        message: update.message,
      });
    }, abortCtrl.signal);
  }

  private async fetchCandidates(query: string, append: boolean): Promise<void> {
    if (!this.workbenchClient.isWorkbenchActive()) {
      return;
    }
    if (this.listStringsAbortController) {
      this.listStringsAbortController.abort();
    }
    const abortCtrl = new AbortController();
    this.listStringsAbortController = abortCtrl;

    const offset = append ? this.rawCandidatesSignal().length : 0;
    this.isLoadingCandidatesSignal.set(true);
    this.errorMessageSignal.set('');

    try {
      const page = await this.workbenchClient.listWorkbenchStrings(
        query,
        offset,
        MaskSmartComponent.PAGE_SIZE,
        abortCtrl.signal,
      );
      if (abortCtrl.signal.aborted) {
        return;
      }
      this.searchRegexErrorSignal.set('');
      if (append) {
        this.rawCandidatesSignal.update((prev) => [...prev, ...page.values]);
      } else {
        this.rawCandidatesSignal.set(page.values);
      }
      this.totalCandidateCountSignal.set(page.totalCount);
    } catch (err) {
      if (abortCtrl.signal.aborted) {
        return;
      }
      if (err instanceof ConnectError && err.code === Code.InvalidArgument) {
        this.searchRegexErrorSignal.set(err.rawMessage || err.message);
        return;
      }
      const detail = err instanceof Error ? err.message : String(err);
      this.errorMessageSignal.set(`Failed to load strings: ${detail}`);
    } finally {
      if (this.listStringsAbortController === abortCtrl) {
        this.isLoadingCandidatesSignal.set(false);
        this.listStringsAbortController = null;
      }
    }
  }

  private cancelInFlightOperations(): void {
    if (this.searchDebounceTimer !== null) {
      clearTimeout(this.searchDebounceTimer);
      this.searchDebounceTimer = null;
    }
    if (this.indexWatchAbortController) {
      this.indexWatchAbortController.abort();
      this.indexWatchAbortController = null;
    }
    if (this.listStringsAbortController) {
      this.listStringsAbortController.abort();
      this.listStringsAbortController = null;
    }
    if (this.exportAbortController) {
      this.exportAbortController.abort();
      this.exportAbortController = null;
    }
  }
}
