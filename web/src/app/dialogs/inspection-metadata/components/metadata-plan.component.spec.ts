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
import { MetadataPlanComponent } from './metadata-plan.component';

describe('MetadataPlanComponent', () => {
  let component: MetadataPlanComponent;
  let fixture: ComponentFixture<MetadataPlanComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataPlanComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MetadataPlanComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('plan', {
      taskGraph: 'digraph G { TaskA -> TaskB; }',
    });
    fixture.detectChanges();
  });

  it('should create and render task graph', () => {
    expect(component).toBeTruthy();
    const element = fixture.nativeElement;
    expect(element.textContent).toContain('digraph G { TaskA -> TaskB; }');
  });
});
