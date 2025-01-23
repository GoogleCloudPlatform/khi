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

import { DiffComponent } from './diff.component';
import { InMemoryWindowConnectionProvider } from 'src/app/services/frame-connection/window-connection-provider.service';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from 'src/app/services/frame-connection/window-connector.service';
import { DiffModule } from 'src/app/diff/diff.module';
import { DiffPageDataSource } from 'src/app/services/frame-connection/frames/diff-page-datasource.service';

describe('DiffComponent', () => {
  let component: DiffComponent;
  let fixture: ComponentFixture<DiffComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiffModule],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        DiffPageDataSource,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
