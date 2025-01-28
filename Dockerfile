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

FROM golang:1.23 as builder

ENV ROOT=/go/src/app
RUN mkdir /built
WORKDIR ${ROOT}
RUN apt update
COPY go.mod go.sum ./
RUN go mod download

COPY . ${ROOT}
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GoogleCloudPlatform/khi/pkg/common/constants.VERSION=$(cat ./VERSION)" -o /built/khi cmd/kubernetes-history-inspector/*.go
RUN mkdir /built/data
COPY ./dist /built/web

FROM scratch
WORKDIR /go/src/app
COPY --from=builder /built /go/src/app
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV GOMEMLIMIT=10000MiB
EXPOSE 8080
ENTRYPOINT [ "/go/src/app/khi" ]
CMD ["--host=0.0.0.0","--temporary-folder=/","--data-destination-folder=/","--frontend-asset-folder=/go/src/app/web"]
