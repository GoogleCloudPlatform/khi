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

import { defineConfig, globalIgnores } from "eslint/config";
import path from "node:path";
import { fileURLToPath } from "node:url";
import js from "@eslint/js";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all
});

export default defineConfig([globalIgnores(["src/app/generated/**/*"]), {
    extends: compat.extends("plugin:storybook/recommended"),
}, {
    files: ["**/*.ts"],

    extends: compat.extends(
        "plugin:@typescript-eslint/recommended",
        "plugin:@angular-eslint/recommended",
        "plugin:@angular-eslint/template/process-inline-templates",
    ),

    languageOptions: {
        ecmaVersion: 5,
        sourceType: "script",

        parserOptions: {
            project: ["tsconfig.json"],
            createDefaultProgram: true,
        },
    },

    rules: {
        "@angular-eslint/directive-selector": ["error", {
            type: "attribute",
            prefix: ["graph", "khi"],
            style: "camelCase",
        }],

        "@angular-eslint/component-selector": ["error", {
            type: "element",
            prefix: ["graph", "khi"],
            style: "kebab-case",
        }],

        "@typescript-eslint/no-unused-vars": "error",
        "@typescript-eslint/no-explicit-any": "error",
    },
}, {
    files: ["**/*.html"],
    extends: compat.extends("plugin:@angular-eslint/template/recommended"),
    rules: {},
}]);
