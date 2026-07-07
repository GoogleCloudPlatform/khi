import { DatePipe } from '@angular/common';
import { Component, computed, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';

import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Displays detailed information about a managed field in a custom tooltip.
 */
@Component({
  selector: 'khi-managed-field-tooltip',
  standalone: true,
  imports: [DatePipe, MatIconModule, KHIIconRegistrationModule],
  templateUrl: './managed-field-tooltip.component.html',
  styleUrl: './managed-field-tooltip.component.scss',
})
export class ManagedFieldTooltipComponent {
  /** Receives the manager name from the annotation provider. */
  readonly manager = input.required<string>();

  /** Receives the timestamp from the annotation provider in nanoseconds. */
  readonly time = input.required<bigint>();

  /** The timezone shift in hours from UTC. */
  readonly timezoneShift = input.required<number>();

  /** The time converted to milliseconds for formatting. */
  protected readonly timeMs = computed(() => {
    const t = this.time();
    if (t === undefined || t === null) return 0;
    try {
      return Number(BigInt(t) / 1000000n);
    } catch {
      return 0;
    }
  });

  /** Converts the timezoneShift into a string format required by DatePipe (e.g. +0900). */
  protected readonly timezoneString = computed(() => {
    const shift = this.timezoneShift();
    if (shift === undefined || shift === null) return '+0000';
    const numShift = Number(shift);
    if (isNaN(numShift)) return '+0000';

    const sign = numShift >= 0 ? '+' : '-';
    const absShift = Math.abs(numShift);
    const hours = Math.floor(absShift).toString().padStart(2, '0');
    const minutes = Math.floor((absShift % 1) * 60)
      .toString()
      .padStart(2, '0');
    return `${sign}${hours}${minutes}`;
  });
}
