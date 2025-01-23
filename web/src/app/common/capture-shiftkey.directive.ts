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

import {
  AfterViewInit,
  Directive,
  EventEmitter,
  HostListener,
  Output,
} from '@angular/core';
import { Subject, distinctUntilChanged } from 'rxjs';

/**
 *
 */
@Directive({
  selector: '[khiCaptureShiftKey]',
})
export class CaptureShiftKeyDirective implements AfterViewInit {
  constructor() {
    this.shiftStatus
      .pipe(distinctUntilChanged())
      .subscribe((status) => this.shiftStatusChange.emit(status));
  }

  @Output() shiftStatusChange = new EventEmitter<boolean>();

  private shiftStatus = new Subject<boolean>();

  private containsMouse = false;

  @HostListener('mouseenter')
  mouseEnter() {
    this.containsMouse = true;
  }

  @HostListener('mouseleave')
  mouseLeave() {
    this.containsMouse = false;
  }

  ngAfterViewInit(): void {
    // the target element may be not able to have focus. In the case, the key related event won't be fired.
    // it will be handled when the event was propagated to the window
    window.addEventListener('keydown', (m) => {
      if (this.containsMouse) this.keyboardEvent(m);
    });
    // to support release shift key not on the target element, it needs to be handled on capture phase.
    // (For in case some element stopping the propagation)
    window.addEventListener('keyup', (m) => this.keyboardEvent(m), {
      capture: true,
    });
  }

  keyboardEvent(m: KeyboardEvent) {
    this.shiftStatus.next(m.shiftKey);
  }
}
