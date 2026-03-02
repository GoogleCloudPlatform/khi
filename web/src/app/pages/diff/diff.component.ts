/**
 * Copyright 2024 Google LLC
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

import { CommonModule } from '@angular/common';
import {
  Component,
  computed,
  EnvironmentInjector,
  inject,
  model,
} from '@angular/core';
import { SideBySideDiffComponent } from 'ngx-diff';
import { map } from 'rxjs';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from 'src/app/annotator/change-pair/resolver';
import { TitleBarComponent } from 'src/app/header/titlebar.component';
import { DiffPageDataSource } from 'src/app/services/frame-connection/frames/diff-page-datasource.service';
import { TimelineLayer } from 'src/app/store/timeline';
import { DiffToolbarComponent } from 'src/app/diff/components/diff-toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import * as yaml from 'js-yaml';
import { toSignal } from '@angular/core/rxjs-interop';
import { CommonFieldAnnotatorComponent } from 'src/app/annotator/common-field-annotator.component';

@Component({
  selector: 'khi-diff-page',
  templateUrl: './diff.component.html',
  styleUrls: ['./diff.component.scss'],
  imports: [
    CommonModule,
    TitleBarComponent,
    SideBySideDiffComponent,
    DiffToolbarComponent,
    CommonFieldAnnotatorComponent,
  ],
})
export class DiffComponent {
  private readonly diffPageSource = inject(DiffPageDataSource);

  private readonly envInjector = inject(EnvironmentInjector);

  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  private readonly changePairAnnotatorResolver = inject(
    CHANGE_PAIR_ANNOTATOR_RESOLVER,
  );

  timeline = toSignal(
    this.diffPageSource.data$.pipe(map((data) => data.timeline)),
    { initialValue: null },
  );

  protected readonly kind = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Kind);
  });

  protected readonly namespace = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Namespace);
  });

  protected readonly name = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Name);
  });

  protected readonly subresource = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Subresource);
  });

  changePair = this.diffPageSource.data$.pipe(
    map((data) => data.timeline.getRevisionPairByLogId(data.logIndex)),
  );

  changePairSignal = toSignal(this.changePair);

  showManagedFields = model(false);

  currentContent = computed(() => {
    const originalContent =
      this.changePairSignal()?.current.resourceContent ?? '';
    if (this.showManagedFields()) {
      return originalContent;
    }
    return this.removeManagedField(originalContent);
  });

  previousContent = computed(() => {
    const originalContent =
      this.changePairSignal()?.previous?.resourceContent ?? '';
    if (this.showManagedFields()) {
      return originalContent;
    }
    return this.removeManagedField(originalContent);
  });

  changePairAnnotators = this.changePairAnnotatorResolver.getResolvedAnnotators(
    this.changePair,
    this.envInjector,
  );

  protected copy(content: string) {
    let snackbarMessage = 'Copy failed';
    if (this.clipboard.copy(content)) {
      snackbarMessage = 'Copied!';
    }
    this.snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }

  private removeManagedField(content: string): string {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const yamlData = yaml.load(content) as any;
      if (
        yamlData &&
        yamlData['metadata'] &&
        yamlData['metadata']['managedFields']
      ) {
        delete yamlData.metadata.managedFields;
      }
      return yamlData ? yaml.dump(yamlData, { lineWidth: -1 }) : content;
    } catch (e) {
      console.warn(`failed to process frontend yaml: ${e}`);
      return content;
    }
  }
}
