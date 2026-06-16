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

import { CommonModule } from '@angular/common';
import { Component, computed, input } from '@angular/core';
import { parse } from 'marked';
import DOMPurify from 'dompurify';

/**
 * Component for rendering and displaying sanitized markdown content inside a popup card.
 */
@Component({
  selector: 'khi-markdown-popup',
  templateUrl: './markdown-popup.component.html',
  styleUrls: ['./markdown-popup.component.scss'],
  imports: [CommonModule],
})
export class MarkdownPopupComponent {
  /**
   * The raw markdown content to render.
   */
  markdown = input.required<string>();

  /**
   * Computed sanitized HTML string generated from the raw markdown input.
   */
  sanitizedHtml = computed<string>(() => {
    const rawHtml = parse(this.markdown(), { async: false }) as string;
    return DOMPurify.sanitize(rawHtml);
  });
}
