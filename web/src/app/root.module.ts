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
  Inject,
  Injector,
  NgModule,
  Optional,
  importProvidersFrom,
  inject,
} from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHighlightOptions } from 'ngx-highlightjs';
import { InspectionDataLoaderService } from './services/data-loader.service';
import { TimelineSelectionService } from './services/timeline-selection.service';
import { InspectionDataStoreService } from './services/inspection-data-store.service';
import { SelectionManagerService } from './services/selection-manager.service';
import { RouterModule, TitleStrategy } from '@angular/router';
import { KHIRoutes } from './app.route';
import { RootComponent } from './root.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from './services/frame-connection/window-connector.service';
import { BroadcastChannelWindowConnectionProvider } from './services/frame-connection/window-connection-provider.service';
import { KHITitleStrategy } from './services/title-strategy.service';
import { MatIconRegistry } from '@angular/material/icon';
import { HttpClientModule } from '@angular/common/http';
import { POPUP_MANAGER } from './services/popup/popup-manager';
import { PopupManagerImpl } from './services/popup/popup-manager-impl';
import { BACKEND_API } from './services/api/backend-api-interface';
import { BackendAPIImpl } from './services/api/backend-api.service';
import { NotificationManager } from './services/notification/notification';
import { ProgressDialogService } from './services/progress/progress-dialog.service';
import {
  BACKEND_CONNECTION,
  BackendConnectionServiceImpl,
} from './services/api/backend-connection.service';
import { DiffPageDataSource } from './services/frame-connection/frames/diff-page-datasource.service';
import { DiffPageDataSourceServer } from './services/frame-connection/frames/diff-page-datasource-server.service';
import { GraphPageDataSourceServer } from './services/frame-connection/frames/graph-page-datasource-server.service';
import {
  KHI_FRONTEND_EXTENSION_BUNDLE,
  KHIExtensionBundle,
} from './extensions/extension-common/extension';
import { environment } from 'src/environments/environment';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from './extensions/extension-common/extension-store';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from './services/timeline-filter.service';
import {
  MAT_TOOLTIP_DEFAULT_OPTIONS,
  MatTooltipDefaultOptions,
} from '@angular/material/tooltip';
import { ViewStateService } from './services/view-state.service';
import { LOG_ANNOTATOR_RESOLVER } from './annotator/log/resolver';
import { getDefaultLogAnnotatorResolver } from './annotator/log/default';
import { LOG_TOOL_ANNOTATOR_RESOLVER } from './annotator/log-tool/resolver';
import { getDefaultLogToolAnnotatorResolver } from './annotator/log-tool/default';
import { TIMELINE_ANNOTATOR_RESOLVER } from './annotator/timeline/resolver';
import { getDefaultTimelineAnnotatorResolver } from './annotator/timeline/default';
import {
  CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
  CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER,
} from './annotator/change-pair-tool/resolver';
import {
  getDefaultChangePairToolAnnotatorResolver,
  getDefaultChangePairToolAnnotatorResolverForFloatingPage,
} from './annotator/change-pair-tool/default';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from './annotator/change-pair/resolver';
import { getDefaultChangePairAnnotatorResolver } from './annotator/change-pair/default';
import { GraphPageDataSource } from './services/frame-connection/frames/graph-page-datasource.service';

@NgModule({
  declarations: [RootComponent],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    RouterModule.forRoot(KHIRoutes),
    // Standoalone components
    environment.pluginModules,
  ],
  providers: [
    { provide: EXTENSION_STORE, useValue: new ExtensionStore() },
    importProvidersFrom(HttpClientModule),
    provideHighlightOptions({
      coreLibraryLoader: () => import('highlight.js/lib/core'),
      lineNumbersLoader: () => import('ngx-highlightjs/line-numbers'),
      languages: {
        yaml: () => import('highlight.js/lib/languages/yaml'),
      },
    }),
    { provide: TitleStrategy, useClass: KHITitleStrategy },
    ...ProgressDialogService.providers(),
    InspectionDataLoaderService,
    DiffPageDataSourceServer,
    GraphPageDataSourceServer,
    GraphPageDataSource,

    TimelineSelectionService,
    InspectionDataStoreService,
    SelectionManagerService,
    WindowConnectorService,
    {
      provide: LOG_ANNOTATOR_RESOLVER,
      useValue: getDefaultLogAnnotatorResolver(),
    },
    {
      provide: LOG_TOOL_ANNOTATOR_RESOLVER,
      useValue: getDefaultLogToolAnnotatorResolver(),
    },
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
    {
      provide: WINDOW_CONNECTION_PROVIDER,
      useValue: new BroadcastChannelWindowConnectionProvider(),
    },
    {
      provide: BACKEND_API,
      useClass: BackendAPIImpl,
    },
    {
      provide: BACKEND_CONNECTION,
      useClass: BackendConnectionServiceImpl,
    },
    {
      provide: POPUP_MANAGER,
      useClass: PopupManagerImpl,
    },
    {
      provide: DEFAULT_TIMELINE_FILTER,
      useFactory: () =>
        new TimelineFilter(
          inject(InspectionDataStoreService),
          inject(ViewStateService),
        ),
    },
    {
      provide: MAT_TOOLTIP_DEFAULT_OPTIONS,
      useValue: {
        disableTooltipInteractivity: true,
        showDelay: 0,
        hideDelay: 0,
      } as MatTooltipDefaultOptions,
    },
    NotificationManager,
    DiffPageDataSource,
  ],
  bootstrap: [RootComponent],
})
export class RootModule {
  constructor(
    injector: Injector,
    @Inject(EXTENSION_STORE) extensionStore: ExtensionStore,
    iconRegistry: MatIconRegistry,
    notificationManager: NotificationManager,
    @Optional()
    @Inject(KHI_FRONTEND_EXTENSION_BUNDLE)
    extensions: KHIExtensionBundle[] | null,
  ) {
    extensionStore.injector = injector;
    if (!extensions) extensions = [];
    iconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    extensions.forEach((extension) => {
      extension.initializeExtension(extensionStore);
    });
    notificationManager.initialize();
  }
}
