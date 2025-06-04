<p style="text-align: center;">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./docs/images/logo-dark.svg">
    <img alt="Kubernetes History Inspector" src="./docs/images/logo-light.svg" width="50%">
  </picture>
</p>

Language: English | [日本語](./README.ja.md)

<hr/>

# Kubernetes History Inspector

Kubernetes History Inspector (KHI) is a rich log visualization tool for Kubernetes clusters. KHI transforms vast quantities of logs into an interactive, comprehensive timeline view.
This makes it an invaluable tool for troubleshooting complex issues that span multiple components within your Kubernetes clusters. Also, KHI is agentless, allowing anyone to access its features without a complicated process.

|Timeline view|Cluster diagram view|
|---|---|
|![Timeline view](./docs/images/timeline.png)|![Cluster diagram](./docs/images/cluster-diagram.png)|
|Timeline view visualizes resource status change timings with timeline charts and manifest diffs from Kubernetes audit logs.|Cluster diagram visualizes relationships among Kubernetes resources, solely from kube-apiserver audit logs.|

## Why use KHI?

### Insightful Log Visualization

The key strength of KHI is its ability to visualize logs of numerous activities associated with each Kubernetes resource as timeline-based graphs, moving beyond traditional text-based log analysis. You do not need to manually filter logs by a single resource and chronologically reading through individual activity logs in text data anymore. Instead, you can grasp what happened at a glance directly from the timeline visualization. Also, in addition to log visualization, KHI allows you to review the raw log data for that specific moment in its familiar log format in text, and even examine the YAML manifests at the time the specific event took place. This significantly simplifies the process of pinpointing the root cause of an event.
KHI can also generate diagrams that depict the state of your Kubernetes cluster's resources and their relationships at a specific point in time. This is invaluable for understanding the status of resources and topology of your cluster at a specific time during an incident.

### Agentless and User friendly

KHI is very easy to set up. It is agentless and allows anyone to easily begin using it without any complicated prior setup on target clusters. Also, KHI enables you to visualize Kubernetes logs through GUI operations. You do not need to write complex queries or commands for log retrieval.
![Feature: quick and easy steps to gather logs](./docs/en/images/feature-query.png)

### Developed from real Log Troubleshooting Experience

KHI is originally developed by the Google Cloud Support team before it became open sourced. It emerged from the practical experience of support engineers, who developed it while analyzing Kubernetes logs in their daily operations. KHI is a tool that takes in their deep expertise in Kubernetes log troubleshooting.

## Supported Products

### Kubernetes cluster

- Google Cloud
  - [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/docs/concepts/kubernetes-engine-overview)
  - [Cloud Composer](https://cloud.google.com/composer/docs/composer-3/composer-overview)
  - [GKE on AWS](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/aws/concepts/architecture)
  - [GKE on Azure](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/azure/concepts/architecture)
  - [GDCV for Baremetal](https://cloud.google.com/kubernetes-engine/distributed-cloud/bare-metal/docs/concepts/about-bare-metal)
  - [GDCV for VMWare](https://cloud.google.com/kubernetes-engine/distributed-cloud/vmware/docs/overview)

- Other
  - kube-apiserver audit logs as JSONlines ([Tutorial](/docs/en/setup-guide/oss-kubernetes-clusters.md))

### Logging backend

- Google Cloud
  - Cloud Logging (For all clusters on Google Cloud)

- Other
  - Log file upload ([Tutorial](/docs/en/setup-guide/oss-kubernetes-clusters.md))

## Getting started

### Run from a docker image

#### Supported environment

- Latest Google Chrome
- `docker` command

> [!IMPORTANT]
> We only test KHI with on the latest version of Google Chrome.
> KHI may work with other browsers, but we do not provide support if it does not.

#### Run KHI

1. Open [Cloud Shell](https://shell.cloud.google.com)
1. Run `docker run -p 127.0.0.1:8080:8080 asia.gcr.io/kubernetes-history-inspector/release:latest`
1. Click the link `http://localhost:8080` on the terminal and start working with KHI!

> [!TIP]
> If you want to run KHI with the other environment where the metadata server is not available,
> you can pass the access token via the program argument.
>
>```bash
>docker run -p 127.0.0.1:8080:8080 asia.gcr.io/kubernetes-history-inspector/release:latest -access-token=`gcloud auth print-access-token`
>```
>

> [!NOTE]
> The container image source may change in the near future. #21

For more details, try [Getting started](/docs/en/tutorial/getting-started.md).

### Run from source code

<details>
<summary>Get Started (Run from source)</summary>

#### Prerequisites

- Go 1.24.*
- Node.js environment 22.13.*
- [`gcloud` CLI](https://cloud.google.com/sdk/docs/install)
- [`jq` command](https://jqlang.org/)

#### Initialization (one-time setup)

1. Download or clone this repository
  e.g. `git clone https://github.com/GoogleCloudPlatform/khi.git`
1. Move to the project root
  e.g. `cd khi`
1. Run `cd ./web && npm install` from the project root

#### Build KHI from source and run

1. [Authorize yourself with `gcloud`](https://cloud.google.com/docs/authentication/gcloud)  
  e.g. `gcloud auth login` if you use your user account credentials
1. Run `make build-web && KHI_FRONTEND_ASSET_FOLDER=./dist go run cmd/kubernetes-history-inspector/main.go` from the project root
  Open `localhost:8080` and start working with KHI!

</details>

> [!IMPORTANT]
> Do not expose KHI port on the internet.
> KHI itself is not providing any authentication or authorization features and KHI is intended to be accessed from its local user.

### Authentication settings

## Settings for Managed Environments

### Google Cloud

#### Permissions

The following permissions are required or recommended.

- **Required**
  - `logging.logEntries.list`
- **Recommended**
  - Permissions to list clusters for cluster type (eg. `container.clusters.list` for GKE)
    This permission is used to show autofill candidates for the log filter. KHI's main functionality is not affected without this permission.
- **Setting**
  - Running KHI on environments with a service account attached, such as Google Cloud Compute Engine Instance: Apply the permissions above to the attached service account.
  - Running KHI locally or on Cloud Shell with a user account: Apply the permissions above to your user account.

> [!WARNING]
> KHI does not respect [ADC](https://cloud.google.com/docs/authentication/provide-credentials-adc) – running KHI on a Compute Engine Instances will always use the attached service account regardless of ADC.
> This specification is subject to change in the future.

#### Audit Logging

- **No required configuration**
  KHI fully works with the default audit logging configuration.
- **Recommended**
  - Kubernetes Engine API Data access audit logs for `DATA_WRITE`

> [!TIP]
> Enabling these will log every patch requests on Pod or Node `.status` field.
> KHI will use this to display detailed container status.
> KHI will still guess the last container status from the audited Pod deletion log even without these logs, however it requires the Pod to be deleted within the queried timeframe.

- **Setup**
  1. In the Google Cloud Console, [go to the Audit Logs](https://console.cloud.google.com/iam-admin/audit) page.
  1. In the Data Access audit logs configuration table, select  `Kubernetes Engine API` from the Service column.
  1. In the Log Types tab, select the `Data write` Data Access audit log type
  1. Click "SAVE".

### OSS Kubernetes

Read [Using KHI with OSS Kubernetes Clusters - Example with Loki](/docs/en/setup-guide/oss-kubernetes-clusters.md).

## User Guide

Read [user guide](/docs/en/visualization-guide/user-guide.md).

## Development Contribution Guide

If you'd like to contribute to the project KHI, read [Contribution Guide](/docs/en/development-contribution/contributing.md) and then follow [Development Guide](/docs/en/development-contribution/development-guide.md)

## Disclaimer

Please note that this tool is not an officially supported Google Cloud product. If you find any issues and have a feature request, [file a Github issue on this repository](https://github.com/GoogleCloudPlatform/khi/issues/new?template=Blank+issue) and we are happy to check them on best-effort basis.
