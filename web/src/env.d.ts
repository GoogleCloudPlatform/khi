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

/**
 * Note: Use ngx-env only for the case that the variable needs to be switch frequently on local development.
 * If the flag is correspoding to the build configuration, use environment files instead.
 */

/* eslint-disable */
declare var process: {
  env: {
    NG_APP_BACKEND_URL_PREFIX: string;
    NG_APP_VERSION: string;
    // Replace the line below with your environment variable for better type checking
    [key: string]: any;
  };
};
