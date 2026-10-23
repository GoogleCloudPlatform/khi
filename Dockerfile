# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.27 AS builder

ENV ROOT=/go/src/app
RUN mkdir /built
WORKDIR ${ROOT}
RUN apt update
COPY go.mod go.sum ./
RUN go mod download

COPY . ${ROOT}
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(cat ./VERSION)" -o /built/khi cmd/kubernetes-history-inspector/*.go
RUN mkdir /built/data
RUN mkdir -m 1777 /app.tmp

FROM scratch AS slim
WORKDIR /go/src/app
COPY --from=builder /built /go/src/app
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app.tmp /tmp

ENV GOMEMLIMIT=10000MiB
ENV HOST=0.0.0.0
ENV KHI_MCP_DISABLE_RENDERER=true

EXPOSE 8080
ENTRYPOINT ["/go/src/app/khi","--temporary-folder=/","--data-destination-folder=/"]

FROM debian:trixie-slim AS default
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    fonts-liberation \
    libgl1 \
    libegl1 \
    libgles2 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/app
COPY --from=builder /built /go/src/app

ENV GOMEMLIMIT=10000MiB
ENV HOST=0.0.0.0
ENV CHROME_BIN=/usr/bin/chromium

EXPOSE 8080
ENTRYPOINT ["/go/src/app/khi","--temporary-folder=/","--data-destination-folder=/"]
