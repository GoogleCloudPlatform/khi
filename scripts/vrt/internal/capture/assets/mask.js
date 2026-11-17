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

function applyKhiVrtMasks(selectors) {
  for (var i = 0; i < selectors.length; i++) {
    var nodes = document.querySelectorAll(selectors[i]);
    for (var j = 0; j < nodes.length; j++) {
      var rect = nodes[j].getBoundingClientRect();
      var mask = document.createElement('div');
      mask.style.position = 'fixed';
      mask.style.left = rect.left + 'px';
      mask.style.top = rect.top + 'px';
      mask.style.width = rect.width + 'px';
      mask.style.height = rect.height + 'px';
      mask.style.background = '#808080';
      mask.style.zIndex = '2147483647';
      mask.style.pointerEvents = 'none';
      document.body.appendChild(mask);
    }
  }
}
