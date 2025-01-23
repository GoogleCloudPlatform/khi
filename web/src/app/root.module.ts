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

import { AppComponent } from './pages/main/main.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHighlightOptions } from 'ngx-highlightjs';
import { InspectionDataLoaderService } from './services/data-loader.service';
import { TimelineSelectionService } from './services/timeline-selection.service';
import { InspectionDataStoreService } from './services/inspection-data-store.service';
import { SelectionManagerService } from './services/selection-manager.service';
import { HeaderModule } from './header/header.module';
import { DialogsModule } from './dialogs/dialogs.module';
import { LogModule } from './log/log.module';
import { DiffModule } from './diff/diff.module';
import { CommonModule } from '@angular/common';
import { TimelineModule } from './timeline/timeline.module';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { RouterModule, TitleStrategy } from '@angular/router';
import { KHIRoutes } from './app.route';
import { RootComponent } from './root.component';
import { GraphPageModule } from './pages/graph/graph.module';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from './services/frame-connection/window-connector.service';
import { BroadcastChannelWindowConnectionProvider } from './services/frame-connection/window-connection-provider.service';
import { KHITitleStrategy } from './services/title-strategy.service';
import { KHICommonModule } from './common/common.module';
import { MatIconModule, MatIconRegistry } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { HttpClientModule } from '@angular/common/http';
import { RequestUserActionPopupComponent } from './dialogs/request-user-action-popup/request-user-action-popup.component';
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
@NgModule({
  declarations: [AppComponent, RootComponent],
  imports: [
    CommonModule,
    KHICommonModule,
    BrowserModule,
    BrowserAnimationsModule,
    HeaderModule,
    DialogsModule,
    LogModule,
    DiffModule,
    GraphPageModule,
    TimelineModule,
    MatSnackBarModule,
    RouterModule.forRoot(KHIRoutes),
    MatIconModule,
    MatButtonModule,
    // Standoalone components
    RequestUserActionPopupComponent,
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

    TimelineSelectionService,
    InspectionDataStoreService,
    SelectionManagerService,
    WindowConnectorService,
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
