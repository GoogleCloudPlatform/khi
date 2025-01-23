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

import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MatDialogModule } from '@angular/material/dialog';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { InspectionDataLoaderService } from '../services/data-loader.service';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { GraphMenuComponent } from './graph-menu.component';

describe('GraphMenuComponent', () => {
  let component: GraphMenuComponent;
  let fixture: ComponentFixture<GraphMenuComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [GraphMenuComponent],
      imports: [MatDialogModule, MatIconModule, MatMenuModule],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        InspectionDataLoaderService,
        InspectionDataStoreService,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(GraphMenuComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
