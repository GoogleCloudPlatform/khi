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
import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { AnnotationDecision } from './annotator';

@Component({
  standalone: true,
  templateUrl: './common-toolbar-button.component.html',
  styleUrl: './common-toolbar-button.component.sass',
  imports: [CommonModule, MatIconModule, MatTooltipModule],
})
export class CommonToolbarButtonComponent {
  @Input()
  public icon = 'content_paste';

  @Input()
  public tooltip = '';

  @Input()
  public onClick = () => {};

  @Input()
  public disabled = false;

  public triggerOnClock() {
    this.onClick();
  }

  public static disabledAnnotationDecision(
    icon: string,
    tooltip: string,
  ): CommonToolbarButtonInput {
    return {
      hidden: true,
      inputs: {
        icon,
        tooltip,
        disabled: true,
        onClick: () => {},
      },
    };
  }
}

export interface CommonToolbarButtonInput extends AnnotationDecision {
  inputs: {
    icon: string;
    tooltip: string;
    disabled: boolean;
    onClick: () => void;
  };
}
