Language: English | [日本語](/docs/ja/setup-guide/google-cloud-permissions.md)

# Google Cloud Permissions & Configuration Guide

Kubernetes History Inspector (KHI) fetches Kubernetes audit logs from Google Cloud Logging. To run inspections on GKE clusters or Google Cloud environments, the identity running KHI requires appropriate IAM permissions and Google Cloud Audit Logging configurations.

## IAM Permissions

### Required & Recommended Permissions

To allow KHI to query Cloud Logging and provide autocomplete candidates in the New Inspection dialog:

* **Required Permission**:
  * `logging.logEntries.list` - Used to query log entries from Cloud Logging.
* **Recommended Permissions**:
  * `monitoring.timeSeries.list` - Used to fetch autocomplete candidates for cluster names and resources in the New Inspection dialog.
  * `container.clusters.list` - Used for cluster metadata discovery when using Cloud Composer features.

### Recommended IAM Roles

Instead of assigning individual permissions, you can assign one of the following standard IAM roles:

| IAM Role | Role ID | Purpose |
| --- | --- | --- |
| **Logs Viewer** | `roles/logging.viewer` | Allows querying standard logs in Cloud Logging. |
| **Private Logs Viewer** | `roles/logging.privateLogViewer` | Allows querying audit logs containing sensitive payload data. (Recommended for full audit log access) |

---

## Authentication Methods

KHI uses Google Cloud Application Default Credentials (ADC) to authenticate API requests to Cloud Logging.

### 1. Cloud Shell (No setup required)

When running KHI in Google Cloud Shell, KHI automatically uses the Cloud Shell default compute metadata service. No additional credential file mounting is needed.

```bash
docker run -p 127.0.0.1:8080:8080 gcr.io/kubernetes-history-inspector/release:latest
```

### 2. Local Environment via User Credentials (gcloud ADC)

When running KHI on your local workstation (Linux, macOS, or Windows), generate ADC credentials using `gcloud` and mount the credentials file into the container:

1. Generate Application Default Credentials:

   ```bash
   gcloud auth application-default login
   ```

2. Run KHI container with mounted credentials:

   **Linux / macOS / WSL:**

   ```bash
   docker run \
     -p 127.0.0.1:8080:8080 \
     -v ~/.config/gcloud/application_default_credentials.json:/root/.config/gcloud/application_default_credentials.json:ro \
     gcr.io/kubernetes-history-inspector/release:latest
   ```

   **Windows PowerShell:**

   ```bash
   docker run `
     -p 127.0.0.1:8080:8080 `
     -v $env:APPDATA\gcloud\application_default_credentials.json:/root/.config/gcloud/application_default_credentials.json:ro `
     gcr.io/kubernetes-history-inspector/release:latest
   ```

### 3. Service Account Key

If running KHI on a VM instance (e.g., GCE) or with a dedicated Service Account, assign the required permissions to the attached service account or mount the key file:

```bash
docker run \
  -p 127.0.0.1:8080:8080 \
  -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/sa-key.json \
  -v /path/to/sa-key.json:/tmp/sa-key.json:ro \
  gcr.io/kubernetes-history-inspector/release:latest
```

### 4. Service Account Impersonation

If you need to impersonate a service account using `gcloud`:

```bash
gcloud auth application-default login --impersonate-service-account=<SERVICE_ACCOUNT_EMAIL>
```

---

## Audit Logging Configuration

### Kubernetes Engine API Audit Logs

* **Default**: KHI fully functions with default Google Cloud audit logging configuration.
* **Recommended**: Enable `DATA_WRITE` Data Access audit logs for Kubernetes Engine API.

> [!TIP]
> Enabling `DATA_WRITE` audit logs records patch requests to Pod or Node `.status` fields.
> KHI uses these logs to display detailed container status histories. If disabled, KHI infers container state from Pod deletion logs, but status changes during Pod execution may not be fully captured.

### Setup Instructions

1. Go to the [Audit Logs page](https://console.cloud.google.com/iam-admin/audit) in the Google Cloud Console.
2. In the Data Access audit logs configuration table, select **Kubernetes Engine API** from the Service column.
3. In the Log Types tab, select the **Data write** Data Access audit log type.
4. Click **Save**.
