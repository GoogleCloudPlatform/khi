# Kubernetes History Inspector

Kubernetes History Inspector (KHI) is a rich visualization tool for Kubernetes clusters on Google Cloud (e.g GKE, GKE on AWS/Azure, GDCV, etc).
KHI transforms vast quantities of logs into an interactive, comprehensive timeline view.
This makes it an invaluable tool for troubleshooting complex issues that span multiple components within your Kubernetes clusters.

![Demo Image](https://github.com/GoogleCloudPlatform/khi/blob/main/image/demo.gif)

## KHI features and characteristics

- **No Prior Setup Required:** KHI uses your existing Cloud Logging logs, so you don't need to install anything extra. This makes setup easier and saves you time. Furthermore, you can use KHI to troubleshoot even past issues as long as logs are still available in Cloud Logging.

- **Effortless log collection:** KHI significantly simplifies the process of collecting and visualizing Kubernetes-related logs. Instead of writing complex queries, users can leverage an interactive GUI. By setting the target cluster type, log types, and parameters such as time range and cluster name, KHI automatically generates the necessary queries and collects the logs for visualization.

![Feature: quick and easy steps to gather logs](./image/feature-query.png)

- **Comprehensive Visualization with Interactive Timelines:** KHI transforms vast quantities of logs into an interactive and comprehensive timeline view.
  - **Resource History Visualization:** KHI displays the status of resources on a timeline. It also parses audit logs and displays the resource manifest at a specific point in time, highlighting differences.
  - **Visualization of Multiple Log Types Across Multiple Resource Types:** KHI correlates various types of logs across related resources, providing a holistic view.
  - **Timeline Comparison of Logs Across Resources:** The timeline view allows users to compare logs across resources in the time dimension, making it easy to identify relationships and dependencies.
  - **Powerful Interactive Filters:** KHI intentionally loads a massive amount of logs into memory. This enables users to interactively filter logs and quickly pinpoint the information they need within the large dataset.

![Feature: timeline view](./image/feature-timeline.png)

- **Cluster Resource Topology Diagrams (Early alpha feature):** KHI can generate diagrams that depict the state of your Kubernetes cluster's resources and their relationships at a specific point in time. This is invaluable for understanding the configuration and topology of your cluster during an incident or for auditing purposes.

![Feature: resource diagram](./image/feature-diagram.png)

## Supported Products

- [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/docs/concepts/kubernetes-engine-overview)
- [Cloud Composer](https://cloud.google.com/composer/docs/composer-3/composer-overview)
- [GKE on AWS](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/aws/concepts/architecture) (Anthos on AWS)
- [GKE on Azure](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/azure/concepts/architecture) (Anthos on Azure)
- [GDCV for Baremetal](https://cloud.google.com/anthos/clusters/docs/bare-metal/1.16/concepts/about-bare-metal) (GKE on Baremtal, Anthos on Baremetal)
- GDCV for VMWare (GKE on VMWare, Anthos on VMWare)

## Get Started

### Prerequisites
- Go 1.21.*
- Node.js environment 18.19.*
- [`gcloud` CLI](https://cloud.google.com/sdk/docs/install)

### Initialization (one-time setup)
1. Download or clone this repository.   
  e.g. `git clone https://github.com/GoogleCloudPlatform/khi.git`
1. Move to the project root.   
  e.g. `cd khi`
1. Run `cd ./web && npm install` from the project root

### Run KHI
1. [Authorize yourself with `gcloud`](https://cloud.google.com/docs/authentication/gcloud).  
  e.g. `gcloud auth login` if you use your user account credentials.
1. Run `make build-web && KHI_FRONTEND_STATIC_FILE_FOLDER=./dist go run cmd/kubernetes-history-inspector/main.go` from the project root.   
  Backend app will run on `localhost:8080` by defaults
1. Run `make watch-web` from the project root.   
  Frontend app will run on `http://localhost:4200` by default

## Examples

// TODO (b/391498707): add examples usage with screenshots

## Contribute

If you'd like to contribute to the project, please read our [Contributing guide](./docs/contributing.md).

## Disclaimer

Please note that this tool is not an officially supported Google Cloud product. If you find any issues and have a feature request, please file a Github issue on this repository and we are happy to check them on best-effort basis.
