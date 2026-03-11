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

import { ClipboardModule, Clipboard } from '@angular/cdk/clipboard';
import { CommonModule } from '@angular/common';
import { Component, input, inject } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'khi-copiable-key-value',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatTooltipModule, ClipboardModule],
  templateUrl: './copiable-key-value.component.html',
  styleUrl: './copiable-key-value.component.scss',
})
export class CopiableKeyValueComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);
  icon = input('');

  label = input('');

  value = input<string>('');

  onValueClick(value: string) {
    let snackbarMessage: string;
    if (this.clipboard.copy(value)) {
      snackbarMessage = 'Copied!';
    } else {
      snackbarMessage = 'Copy failed.';
    }
    this.snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }
}
