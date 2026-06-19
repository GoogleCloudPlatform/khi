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

module.exports = {
  extends: ["stylelint-config-sass-guidelines"],
  rules: {
    "max-nesting-depth": 5,
    "color-named": null,
    "selector-pseudo-element-no-unknown": [
      true,
      { ignorePseudoElements: ["ng-deep"] },
    ],
    // Indentation is handled by Prettier, so it is disabled here to avoid conflicts.
    "@stylistic/indentation": null,
  },
  overrides: [
    {
      files: ["**/golden-layout-khi-theme.scss"],
      rules: {
        "selector-class-pattern": null,
        "max-nesting-depth": null,
      },
    },
  ],
};
