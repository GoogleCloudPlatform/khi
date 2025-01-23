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

import { TestBed } from '@angular/core/testing';
import { MatDialogModule } from '@angular/material/dialog';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { AppComponent } from './main.component';
import { DiffModule } from '../../diff/diff.module';
import { HeaderModule } from '../../header/header.module';
import { LogModule } from '../../log/log.module';
import { InspectionDataLoaderService } from '../../services/data-loader.service';
import { TimelineModule } from '../../timeline/timeline.module';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from 'src/app/services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from 'src/app/services/frame-connection/window-connection-provider.service';
import { MatIconModule } from '@angular/material/icon';
import { provideHttpClient } from '@angular/common/http';
import { POPUP_MANAGER } from 'src/app/services/popup/popup-manager';
import { MockPopupManager } from 'src/app/services/popup/mock';
import { DiffPageDataSourceServer } from 'src/app/services/frame-connection/frames/diff-page-datasource-server.service';
import { GraphPageDataSourceServer } from 'src/app/services/frame-connection/frames/graph-page-datasource-server.service';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from 'src/app/services/timeline-filter.service';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { ViewStateService } from 'src/app/services/view-state.service';

describe('AppComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [AppComponent],
      imports: [
        MatDialogModule,
        HeaderModule,
        TimelineModule,
        LogModule,
        DiffModule,
        MatIconModule,
        MatSnackBarModule,
      ],
      providers: [
        {
          provide: EXTENSION_STORE,
          useValue: new ExtensionStore(),
        },
        InspectionDataLoaderService,
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: POPUP_MANAGER,
          useValue: new MockPopupManager(),
        },
        {
          provide: DEFAULT_TIMELINE_FILTER,
          useValue: new TimelineFilter(
            new InspectionDataStoreService(),
            new ViewStateService(),
          ),
        },
        provideHttpClient(),
        DiffPageDataSourceServer,
        GraphPageDataSourceServer,
      ],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });
});
