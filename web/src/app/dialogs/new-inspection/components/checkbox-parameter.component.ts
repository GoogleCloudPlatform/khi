import { CommonModule } from '@angular/common';
import { Component, computed, inject, input } from '@angular/core';
import { MatCheckboxModule } from '@angular/material/checkbox';
import {
  CheckboxParameterFormField,
  ParameterHintType,
} from 'src/app/common/schema/form-types';
import { ParameterHeaderComponent } from './parameter-header.component';
import { ParameterHintComponent } from './parameter-hint.component';
import { PARAMETER_STORE } from './service/parameter-store';

/**
 * A form field for checkbox type parameter in the new-inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-checkbox-parameter',
  templateUrl: './checkbox-parameter.component.html',
  styleUrls: ['./checkbox-parameter.component.scss'],
  imports: [
    CommonModule,
    MatCheckboxModule,
    ParameterHeaderComponent,
    ParameterHintComponent,
  ],
})
export class CheckboxParameterComponent {
  /**
   * Exposes ParameterHintType enum to the template.
   */
  protected readonly ParameterHintType = ParameterHintType;

  /**
   * The spec of this checkbox type parameter.
   */
  readonly parameter = input.required<CheckboxParameterFormField>();

  /**
   * Injects the PARAMETER_STORE service.
   */
  private readonly store = inject(PARAMETER_STORE);

  /**
   * Computed checked state of the checkbox parameter.
   */
  readonly isChecked = computed(() => {
    return (
      this.store.get<boolean>(this.parameter().id)() ?? this.parameter().default
    );
  });

  /**
   * Handles toggle change events from the checkbox.
   */
  onToggle(checked: boolean): void {
    if (!this.parameter().readonly) {
      this.store.set(this.parameter().id, checked);
    }
  }
}
