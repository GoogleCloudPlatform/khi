/**
 * Copyright 2026 Google LLC
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
  enableProdMode,
  provideZoneChangeDetection,
  inject,
  provideAppInitializer,
  Injector,
  ErrorHandler,
  importProvidersFrom,
  ApplicationConfig,
} from '@angular/core';
import { ReporterErrorHandler } from './app/common/reporter/reporter-error-handler';
import { bootstrapApplication } from '@angular/platform-browser';
import { RootComponent } from './app/root.component';
import { environment } from './environments/environment';
import { provideRouter } from '@angular/router';
import { KHIRoutes } from './app/app.route';
import { provideHttpClient } from '@angular/common/http';
import { provideHighlightOptions } from 'ngx-highlightjs';
import { TitleStrategy } from '@angular/router';
import { KHITitleStrategy } from './app/services/title-strategy.service';
import { ProgressDialogService } from './app/services/progress/progress-dialog.service';
import { InspectionDataLoaderService } from './app/services/data-loader.service';
import { DiffPageDataSourceServer } from './app/services/frame-connection/frames/diff-page-datasource-server.service';
import { GraphPageDataSourceServer } from './app/services/frame-connection/frames/graph-page-datasource-server.service';
import { GraphPageDataSource } from './app/services/frame-connection/frames/graph-page-datasource.service';
import { TimelineSelectionService } from './app/services/timeline-selection.service';
import { InspectionDataStoreService } from './app/services/inspection-data-store.service';
import { SelectionManagerService } from './app/services/selection-manager.service';
import {
  WindowConnectorService,
  WINDOW_CONNECTION_PROVIDER,
} from './app/services/frame-connection/window-connector.service';
import { BroadcastChannelWindowConnectionProvider } from './app/services/frame-connection/window-connection-provider.service';
import { BACKEND_API } from './app/services/api/backend-api-interface';
import { BackendAPIImpl } from './app/services/api/backend-api.service';
import {
  BACKEND_SYNC,
  BackendSyncServiceImpl,
} from './app/services/api/backend-sync.service';
import { POPUP_MANAGER } from './app/services/popup/popup-manager';
import { PopupManagerImpl } from './app/services/popup/popup-manager-impl';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from './app/services/timeline-filter.service';
import { ViewStateService } from './app/services/view-state.service';
import {
  MAT_TOOLTIP_DEFAULT_OPTIONS,
  MatTooltipDefaultOptions,
} from '@angular/material/tooltip';
import {
  FILE_UPLOADER,
  KHIServerFileUploader,
} from './app/dialogs/new-inspection/components/service/file-uploader';
import { NotificationManager } from './app/services/notification/notification';
import { DiffPageDataSource } from './app/services/frame-connection/frames/diff-page-datasource.service';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from './app/extensions/extension-common/extension-store';
import { KHI_FRONTEND_EXTENSION_BUNDLES } from './app/extensions/extension-common/extension';
import { provideAnimations } from '@angular/platform-browser/animations';
import { KHIIconRegistrationModule } from './app/shared/module/icon-registration.module';

if (environment.production) {
  enableProdMode();
}

/**
 * Application configuration for KHI.
 * Defines providers used during bootstrap.
 */
export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection(),
    provideRouter(KHIRoutes),
    provideAnimations(),
    { provide: EXTENSION_STORE, useValue: new ExtensionStore() },
    provideHttpClient(),
    provideHighlightOptions({
      coreLibraryLoader: () => import('highlight.js/lib/core'),
      lineNumbersLoader: () => import('ngx-highlightjs/line-numbers'),
      languages: {
        yaml: () => import('highlight.js/lib/languages/yaml'),
      },
    }),
    { provide: TitleStrategy, useClass: KHITitleStrategy },
    { provide: ErrorHandler, useClass: ReporterErrorHandler },
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
      provide: WINDOW_CONNECTION_PROVIDER,
      useValue: new BroadcastChannelWindowConnectionProvider(),
    },
    {
      provide: BACKEND_API,
      useClass: BackendAPIImpl,
    },
    {
      provide: BACKEND_SYNC,
      useClass: BackendSyncServiceImpl,
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
    {
      provide: FILE_UPLOADER,
      useClass: KHIServerFileUploader,
    },
    NotificationManager,
    DiffPageDataSource,
    importProvidersFrom(KHIIconRegistrationModule),
    importProvidersFrom(...environment.pluginModules),
    provideAppInitializer(() => {
      const extensionStore = inject(EXTENSION_STORE);
      const notificationManager = inject(NotificationManager);
      const extensions =
        inject(KHI_FRONTEND_EXTENSION_BUNDLES, { optional: true }) || [];
      const injector = inject(Injector);

      extensionStore.injector = injector;
      extensions.forEach((extension) => {
        extension.initializeExtension(extensionStore);
      });
      notificationManager.initialize();
    }),
  ],
};

bootstrapApplication(RootComponent, appConfig).catch((err) =>
  console.error(err),
);
