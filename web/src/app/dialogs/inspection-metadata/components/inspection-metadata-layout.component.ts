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

import { Component, input, output, viewChild, computed } from '@angular/core';
import { MatAccordion, MatExpansionModule } from '@angular/material/expansion';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { JobCommandComponent } from 'src/app/dialogs/new-inspection/components/job-command.component';
import { InspectionMetadataViewModel } from '../types/inspection-metadata.model';
import { MetadataOverviewComponent } from './metadata-overview.component';
import { MetadataErrorsComponent } from './metadata-errors.component';
import { MetadataQueriesComponent } from './metadata-queries.component';
import { MetadataLogsComponent } from './metadata-logs.component';
import { MetadataPlanComponent } from './metadata-plan.component';

/**
 * Dumb layout component for the Inspection Metadata dialog.
 * Organizes metadata sections into an accordion with toolbar controls and dialog header.
 */
@Component({
  selector: 'khi-inspection-metadata-layout',
  imports: [
    MatExpansionModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
    JobCommandComponent,
    MetadataOverviewComponent,
    MetadataErrorsComponent,
    MetadataQueriesComponent,
    MetadataLogsComponent,
    MetadataPlanComponent,
  ],
  templateUrl: './inspection-metadata-layout.component.html',
  styleUrls: ['./inspection-metadata-layout.component.scss'],
})
export class InspectionMetadataLayoutComponent {
  /** The aggregated metadata view model. */
  readonly viewModel = input.required<InspectionMetadataViewModel>();

  /** Emitted when the user clicks the dialog close button. */
  readonly closed = output<void>();

  /** Reference to the underlying Angular Material accordion. */
  private readonly accordion = viewChild<MatAccordion>('accordion');

  /** Whether the error panel should be displayed. */
  protected readonly hasErrors = computed(
    () => this.viewModel().errors.length > 0,
  );

  /** Expands all accordion panels. */
  protected expandAll(): void {
    this.accordion()?.openAll();
  }

  /** Collapses all accordion panels. */
  protected collapseAll(): void {
    this.accordion()?.closeAll();
  }
}
