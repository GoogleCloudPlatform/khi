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

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TimelineComponent } from './timeline.component';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { OverlayModule } from '@angular/cdk/overlay';
import { MatIconModule } from '@angular/material/icon';
import { KHICommonModule } from '../common/common.module';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NgxEnvModule } from '@ngx-env/core';
import { NavigatorComponent } from './navigator/navigator.component';

@NgModule({
  declarations: [TimelineComponent, NavigatorComponent],
  imports: [
    CommonModule,
    KHICommonModule,
    ScrollingModule,
    OverlayModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    NgxEnvModule,
  ],
  exports: [TimelineComponent],
})
export class TimelineModule {}
