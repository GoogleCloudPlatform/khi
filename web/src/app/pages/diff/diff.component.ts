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
import { Component, computed, inject, model } from '@angular/core';
import { SideBySideDiffComponent } from 'ngx-diff';
import { map } from 'rxjs';
import { HeaderSmartComponent } from 'src/app/header/header-smart.component';
import { DiffPageDataSource } from 'src/app/services/frame-connection/frames/diff-page-datasource.service';
import { DiffToolbarComponent } from 'src/app/diff/components/diff-toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import * as yaml from 'js-yaml';
import { toSignal } from '@angular/core/rxjs-interop';
import { CopiableKeyValueComponent } from 'src/app/shared/components/copiable-key-value/copiable-key-value.component';

@Component({
  selector: 'khi-diff-page',
  templateUrl: './diff.component.html',
  styleUrls: ['./diff.component.scss'],
  imports: [
    CommonModule,
    HeaderSmartComponent,
    SideBySideDiffComponent,
    DiffToolbarComponent,
    CopiableKeyValueComponent,
  ],
})
export class DiffComponent {
  private readonly diffPageSource = inject(DiffPageDataSource);

  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  timelinePath = toSignal(
    this.diffPageSource.data$.pipe(map((data) => data.timelinePath)),
    { initialValue: [] },
  );

  protected readonly pathNodes = computed(() => {
    return this.timelinePath().map((node) => ({
      label: node.type.label,
      value: node.label,
      icon: node.type.icon || 'label',
    }));
  });

  private readonly data = toSignal(this.diffPageSource.data$, {
    initialValue: null,
  });

  showManagedFields = model(false);

  currentContent = computed(() => {
    const originalContent = this.data()?.currentContent ?? '';
    if (this.showManagedFields()) {
      return originalContent;
    }
    return this.removeManagedField(originalContent);
  });

  previousContent = computed(() => {
    const originalContent = this.data()?.previousContent ?? '';
    if (this.showManagedFields()) {
      return originalContent;
    }
    return this.removeManagedField(originalContent);
  });

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
