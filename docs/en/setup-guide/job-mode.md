# Job Mode Guide (For CI/CD and Automation)

KHI includes a **Job mode** that performs log analysis and generates a `.khi` file directly at a specified path without starting the web server.

Job mode is useful for automated workflows, such as generating `.khi` files when alerts are triggered or capturing inspection snapshots during CI/CD pipeline runs (deployments, tests, etc.). The generated `.khi` file can later be uploaded to the KHI Web UI for interactive analysis.

## Obtaining Job Mode Commands from the Web UI

When you fill out the parameters on the "New Inspection" page in the KHI Web UI, a Job mode CLI command representing those parameters is generated at the bottom of the form.

![Job Mode in KHI UI](../../images/job-mode.png)

> [!NOTE]
> The CLI command displayed in the UI uses a direct binary execution format (e.g., `./khi ...`). When running via Docker, mount the output directory as shown below.

## Running in Docker Containers

Mount the output directory into the container (e.g., `-v $(pwd):/output`):

```bash
docker run --rm \
  -v $(pwd):/output \
  gcr.io/kubernetes-history-inspector/release:latest \
  --job-mode \
  --job-inspection-type="gke-basic" \
  --job-inspection-features="ALL" \
  --job-inspection-values='{"projectId":"my-gcp-project","clusterName":"my-cluster","location":"us-central1"}' \
  --job-export-destination="/output/result.khi"
```

> [!IMPORTANT]
> **Replacing File Placeholders and Mounting Input Files**
>
> When the inspection parameters include local files (such as uploaded log files), the generated command contains `"path/to/file"` as a placeholder in `--job-inspection-values`.
> When executing via Docker, replace `"path/to/file"` with the actual input file path mounted inside the container:
>
> ```bash
> docker run --rm \
>   -v $(pwd):/output \
>   -v /path/to/local/audit.log:/input/audit.log:ro \
>   gcr.io/kubernetes-history-inspector/release:latest \
>   --job-mode \
>   --job-inspection-type="oss-log" \
>   --job-inspection-features="ALL" \
>   --job-inspection-values='{"logFilePath":"/input/audit.log"}' \
>   --job-export-destination="/output/result.khi"
> ```

## Parameter Details

For the definitions and specifications of all parameters used in Job mode, see [pkg/parameters/job.go](../../../pkg/parameters/job.go).
