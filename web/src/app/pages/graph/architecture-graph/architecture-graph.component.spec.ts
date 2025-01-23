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

import { ArchitectureGraphComponent } from './architecture-graph.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../../../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../../../services/frame-connection/window-connection-provider.service';
import { GraphPageDataSource } from 'src/app/services/frame-connection/frames/graph-page-datasource.service';

describe('ArchitectureGraphComponent', () => {
  let component: ArchitectureGraphComponent;
  let fixture: ComponentFixture<ArchitectureGraphComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ArchitectureGraphComponent],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        GraphPageDataSource,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ArchitectureGraphComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
