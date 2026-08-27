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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { GraphSmartComponent } from 'src/app/graph/graph-smart.component';
import { GraphConverterService } from 'src/app/services/graph-converter.service';
import { emptyGraphData } from 'src/app/common/schema/graph-schema';

describe('GraphSmartComponent', () => {
  let component: GraphSmartComponent;
  let fixture: ComponentFixture<GraphSmartComponent>;
  let mockGraphConverter: jasmine.SpyObj<GraphConverterService>;

  beforeEach(async () => {
    mockGraphConverter = jasmine.createSpyObj('GraphConverterService', [
      'getGraphDataAt',
    ]);
    mockGraphConverter.getGraphDataAt.and.resolveTo(emptyGraphData());

    await TestBed.configureTestingModule({
      imports: [GraphSmartComponent],
      providers: [
        { provide: GraphConverterService, useValue: mockGraphConverter },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(GraphSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
