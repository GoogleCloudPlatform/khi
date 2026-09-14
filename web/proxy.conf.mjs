/**
 * Copyright 2025 Google LLC
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
 * Angular Proxy Configuration
 *
 * This file defines the proxy configuration for the Angular development server.
 * During development, requests to the /api/ path are forwarded to the backend server.
 */

const backendPort = process.env.BACKEND_PORT || process.env.PORT || "8080";
const rawHost = process.env.BACKEND_HOST || process.env.HOST || "127.0.0.1";
const backendHost = rawHost === "0.0.0.0" ? "127.0.0.1" : rawHost;

export default {
  "/api": {
    target: `http://${backendHost}:${backendPort}`,
    secure: false,
    changeOrigin: true,
    logLevel: "debug",
  },
};
