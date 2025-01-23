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

import { NgModule } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import { HeaderComponent } from './header.component';
import { ToolbarComponent } from './toolbar.component';
import { NgxEnvModule } from '@ngx-env/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RegexInputComponent } from './regex-input.component';
import { SetInputComponent } from './set-input.component';
import { OverlayModule } from '@angular/cdk/overlay';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatChipsModule } from '@angular/material/chips';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { KHICommonModule } from '../common/common.module';
import { CommonModule } from '@angular/common';
import { TitleBarComponent } from './titlebar.component';
import { MatMenuModule } from '@angular/material/menu';
import { MainMenuComponent } from './main-menu.component';
import { GraphMenuComponent } from './graph-menu.component';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatButtonToggleModule } from '@angular/material/button-toggle';

@NgModule({
  declarations: [
    HeaderComponent,
    ToolbarComponent,
    SetInputComponent,
    RegexInputComponent,
    TitleBarComponent,
    MainMenuComponent,
    GraphMenuComponent,
  ],
  imports: [
    CommonModule,
    KHICommonModule,
    MatButtonModule,
    MatIconModule,
    MatToolbarModule,
    MatFormFieldModule,
    MatAutocompleteModule,
    MatChipsModule,
    MatInputModule,
    MatMenuModule,
    ReactiveFormsModule,
    FormsModule,
    OverlayModule,
    NgxEnvModule,
    MatTooltipModule,
    MatButtonToggleModule,
  ],
  exports: [HeaderComponent, TitleBarComponent, GraphMenuComponent],
})
export class HeaderModule {}
