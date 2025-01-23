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
import { MetaTableRowComponent } from './meta-table-row.component';
import { ClipboardModule } from '@angular/cdk/clipboard';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

describe('MetaTableRowComponent', () => {
  let fixture: ComponentFixture<MetaTableRowComponent>;
  let component: MetaTableRowComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [MetaTableRowComponent],
      imports: [MatIconModule, MatTooltipModule, ClipboardModule],
    });

    fixture = TestBed.createComponent(MetaTableRowComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should show the key given from Input', () => {
    component.key = 'key';
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.key')?.textContent).toContain('key');
  });

  it('should show the value given from Input', () => {
    component.value = 'value';
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.value-inner')?.textContent).toContain(
      'value',
    );
  });

  it('should show icon only when icon is given from Input', () => {
    component.icon = '';
    const compiled = fixture.nativeElement as HTMLElement;
    fixture.detectChanges();
    expect(compiled.querySelector('.icon')).toBeNull();

    component.icon = 'home';
    fixture.detectChanges();
    expect(compiled.querySelector('.icon')).not.toBeNull();
    expect(compiled.querySelector('.icon')?.textContent).toContain('home');
  });
});
