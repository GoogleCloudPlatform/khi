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

import { Routes } from '@angular/router';
import { AppComponent } from './pages/main/main.component';
import { GraphComponent } from './pages/graph/graph.component';
import {
  DiffPageDeactivateGuard,
  DiffPageGuard,
  GraphPageDeactiveGuard,
  GraphPageGuard,
  PageOpenLifecycleGuard,
  SessionChildGuard,
  SessionDeactivateGuard,
  SessionHostGuard,
} from './app.route.guard';
import { DiffComponent } from './pages/diff/diff.component';
import { PageType } from './extensions/extension-common/extension-types/lifecycle-hook';

export const KHIRoutes: Routes = [
  { path: '', redirectTo: 'session/0', pathMatch: 'full' },
  {
    path: 'session/:sessionId',
    component: AppComponent,
    title: 'KHI - Main view',
    canActivate: [SessionHostGuard, PageOpenLifecycleGuard(PageType.Main)],
    canDeactivate: [SessionDeactivateGuard],
  },
  {
    path: 'session/:sessionId/graph',
    component: GraphComponent,
    title: 'KHI - Graph view',
    canActivate: [
      SessionChildGuard('Diagram'),
      GraphPageGuard,
      PageOpenLifecycleGuard(PageType.GraphView),
    ],
    canDeactivate: [SessionDeactivateGuard, GraphPageDeactiveGuard],
  },
  {
    path: 'session/:sessionId/diff/:kind/:namespace/:resourceName/:subresource',
    component: DiffComponent,
    title: 'KHI - Diff view',
    canActivate: [
      SessionChildGuard('Diff'),
      DiffPageGuard,
      PageOpenLifecycleGuard(PageType.DiffView),
    ],
    canDeactivate: [SessionDeactivateGuard, DiffPageDeactivateGuard],
  },
  { path: '**', redirectTo: '/' },
];
