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
import { DiffViewComponent } from './diff-view.component';
import { LogModule } from '../log/log.module';
import { ParsePrincipalPipe } from './diff-view-pipes';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import { ClipboardModule } from '@angular/cdk/clipboard';
import { KHICommonModule } from '../common/common.module';
import { MatButtonModule } from '@angular/material/button';
import { CommonModule } from '@angular/common';
import { DiffComponent } from '../pages/diff/diff.component';
import { HeaderModule } from '../header/header.module';
import { HighlightModule, provideHighlightOptions } from 'ngx-highlightjs';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { UnifiedDiffComponent, SideBySideDiffComponent } from 'ngx-diff';
import { TIMELINE_ANNOTATOR_RESOLVER } from '../annotator/timeline/resolver';
import { getDefaultTimelineAnnotatorResolver } from '../annotator/timeline/default';
import {
  CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
  CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER,
} from '../annotator/change-pair-tool/resolver';
import {
  getDefaultChangePairToolAnnotatorResolver,
  getDefaultChangePairToolAnnotatorResolverForFloatingPage,
} from '../annotator/change-pair-tool/default';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from '../annotator/change-pair/resolver';
import { getDefaultChangePairAnnotatorResolver } from '../annotator/change-pair/default';

@NgModule({
  declarations: [DiffViewComponent, ParsePrincipalPipe, DiffComponent],
  imports: [
    UnifiedDiffComponent,
    SideBySideDiffComponent,
    CommonModule,
    KHICommonModule,
    LogModule,
    MatIconModule,
    MatToolbarModule,
    ClipboardModule,
    MatButtonModule,
    HeaderModule,
    HighlightModule,
    ScrollingModule,
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
      provide: TIMELINE_ANNOTATOR_RESOLVER,
      useValue: getDefaultTimelineAnnotatorResolver(),
    },
    {
      provide: CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER,
      useValue: getDefaultChangePairToolAnnotatorResolver(),
    },
    {
      provide: CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
      useValue: getDefaultChangePairToolAnnotatorResolverForFloatingPage(),
    },
    {
      provide: CHANGE_PAIR_ANNOTATOR_RESOLVER,
      useValue: getDefaultChangePairAnnotatorResolver(),
    },
  ],
  exports: [DiffViewComponent],
})
export class DiffModule {}
