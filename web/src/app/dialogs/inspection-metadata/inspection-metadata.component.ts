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

import { Component, computed, inject } from '@angular/core';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogConfig,
  MatDialogRef,
} from '@angular/material/dialog';
import { InspectionMetadataOfRunResult } from 'src/app/common/schema/api-types';
import { convertToInspectionMetadataViewModel } from './types/inspection-metadata.model';
import { InspectionMetadataLayoutComponent } from './components/inspection-metadata-layout.component';

/**
 * Smart dialog component for displaying inspection metadata.
 * Bridges raw backend metadata received via MAT_DIALOG_DATA to the layout component.
 */
@Component({
  templateUrl: './inspection-metadata.component.html',
  styleUrls: ['./inspection-metadata.component.scss'],
  imports: [InspectionMetadataLayoutComponent],
})
export class InspectionMetadataDialogComponent {
  /** The raw metadata passed through dialog data. */
  readonly rawMetadata = inject<InspectionMetadataOfRunResult>(MAT_DIALOG_DATA);

  /** Reference to the dialog instance. */
  readonly dialogRef = inject(MatDialogRef<InspectionMetadataDialogComponent>);

  /** View model transformed for presentation. */
  readonly vm = computed(() =>
    convertToInspectionMetadataViewModel(this.rawMetadata),
  );

  /** Closes the inspection metadata dialog. */
  close(): void {
    this.dialogRef.close();
  }
}

/**
 * Opens the Inspection Metadata dialog with standard dimensions and configuration.
 * @param dialog MatDialog service instance.
 * @param data Inspection metadata result to display.
 * @param config Optional dialog configuration overrides.
 * @returns MatDialogRef for the opened dialog.
 */
export function openInspectionMetadataDialog(
  dialog: MatDialog,
  data: InspectionMetadataOfRunResult,
  config: Partial<MatDialogConfig> = {},
): MatDialogRef<InspectionMetadataDialogComponent> {
  return dialog.open(InspectionMetadataDialogComponent, {
    maxWidth: '95vw',
    width: '900px',
    maxHeight: '85vh',
    data,
    ...config,
  });
}
