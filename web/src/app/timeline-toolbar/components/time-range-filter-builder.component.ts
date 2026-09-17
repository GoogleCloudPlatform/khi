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
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  linkedSignal,
  output,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimeRangeFilter } from 'src/app/services/view-state.service';
import {
  formatIsoTimestampNs,
  parseIsoTimestampNs,
} from 'src/app/utils/time-format-util';

/**
 * Dumb popover builder component for manually configuring a time range filter.
 */
@Component({
  selector: 'khi-time-range-filter-builder',
  templateUrl: './time-range-filter-builder.component.html',
  styleUrls: ['./time-range-filter-builder.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    MatInputModule,
    MatFormFieldModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
})
export class TimeRangeFilterBuilderComponent {
  /** The current filter start timestamp in nanoseconds, or null if unset. */
  readonly startTime = input<bigint | null>(null);

  /** The current filter end timestamp in nanoseconds, or null if unset. */
  readonly endTime = input<bigint | null>(null);

  /** The default start timestamp in nanoseconds (e.g. inspection start time). */
  readonly defaultStartTime = input<bigint>(0n);

  /** The default end timestamp in nanoseconds (e.g. inspection end time). */
  readonly defaultEndTime = input<bigint>(0n);

  /** Timezone shift in hours relative to UTC. */
  readonly timezoneShift = input<number>(0);

  /** Whether to show the delete/clear button. */
  readonly showDeleteButton = input<boolean>(false);

  /** Emitted when the cancel/close button is clicked. */
  readonly closeButtonClicked = output<void>();

  /** Emitted when the delete/clear button is clicked. */
  readonly deleteButtonClicked = output<void>();

  /** Emitted when the time range is confirmed and applied. */
  readonly confirm = output<TimeRangeFilter>();

  // Internal state
  protected readonly startInputText = linkedSignal({
    source: () => ({
      start: this.startTime() ?? this.defaultStartTime(),
      shift: this.timezoneShift(),
    }),
    computation: ({ start, shift }) =>
      start > 0n ? formatIsoTimestampNs(start, shift) : '',
  });

  protected readonly endInputText = linkedSignal({
    source: () => ({
      end: this.endTime() ?? this.defaultEndTime(),
      shift: this.timezoneShift(),
    }),
    computation: ({ end, shift }) =>
      end > 0n ? formatIsoTimestampNs(end, shift) : '',
  });

  protected readonly parsedStart = computed(() =>
    parseIsoTimestampNs(this.startInputText(), this.timezoneShift()),
  );
  protected readonly parsedEnd = computed(() =>
    parseIsoTimestampNs(this.endInputText(), this.timezoneShift()),
  );

  protected readonly validationError = computed(() => {
    if (!this.startInputText().trim() || !this.endInputText().trim()) {
      return 'Start and end time are required.';
    }
    const start = this.parsedStart();
    if (start === null) {
      return 'Invalid start time format.';
    }
    const end = this.parsedEnd();
    if (end === null) {
      return 'Invalid end time format.';
    }
    if (start > end) {
      return 'Start time must be before or equal to end time.';
    }
    return '';
  });

  protected readonly isConfirmDisabled = computed(
    () => this.validationError() !== '',
  );

  protected onConfirm(): void {
    if (!this.isConfirmDisabled()) {
      this.confirm.emit({
        startTime: this.parsedStart()!,
        endTime: this.parsedEnd()!,
      });
    }
  }
}
