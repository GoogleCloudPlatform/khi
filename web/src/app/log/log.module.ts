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
import { LogBodyComponent } from './body.component';
import { LogHeaderComponent } from './header.component';
import { LogViewComponent } from './log-view.component';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { ClipboardModule } from '@angular/cdk/clipboard';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { HighlightModule, provideHighlightOptions } from 'ngx-highlightjs';
import { KHICommonModule } from '../common/common.module';
import { CommonModule, NgComponentOutlet } from '@angular/common';
import { IconToggleButtonComponent } from './icon-toggle-button.component';
import { MatTooltipModule } from '@angular/material/tooltip';
import { LOG_ANNOTATOR_RESOLVER } from '../annotator/log/resolver';
import { HighlightLineNumbers } from 'ngx-highlightjs/line-numbers';
import { getDefaultLogAnnotatorResolver } from '../annotator/log/default';
import { LOG_TOOL_ANNOTATOR_RESOLVER } from '../annotator/log-tool/resolver';
import { getDefaultLogToolAnnotatorResolver } from '../annotator/log-tool/default';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { LogViewLogLineComponent } from './log-view-log-line.component';

@NgModule({
  declarations: [
    LogBodyComponent,
    LogHeaderComponent,
    LogViewComponent,
    LogViewLogLineComponent,
    IconToggleButtonComponent,
  ],
  imports: [
    CommonModule,
    KHICommonModule,
    MatToolbarModule,
    MatIconModule,
    MatButtonModule,
    MatSlideToggleModule,
    FormsModule,
    ReactiveFormsModule,
    ScrollingModule,
    HighlightModule,
    HighlightLineNumbers,
    ClipboardModule,
    MatTooltipModule,
    NgComponentOutlet,
    MatSnackBarModule,
  ],
  providers: [
    provideHighlightOptions({
      coreLibraryLoader: () => import('highlight.js/lib/core'),
      lineNumbersLoader: () => import('ngx-highlightjs/line-numbers'),
      languages: {
        yaml: () => import('highlight.js/lib/languages/yaml'),
      },
    }),
    {
      provide: LOG_ANNOTATOR_RESOLVER,
      useValue: getDefaultLogAnnotatorResolver(),
    },
    {
      provide: LOG_TOOL_ANNOTATOR_RESOLVER,
      useValue: getDefaultLogToolAnnotatorResolver(),
    },
  ],
  exports: [LogViewComponent, LogHeaderComponent],
})
export class LogModule {}
