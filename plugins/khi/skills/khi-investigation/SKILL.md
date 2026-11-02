---
name: khi-investigation
description: Investigates Kubernetes cluster incidents and diagnoses unexpected Pod, Node, or resource behavior using Kubernetes History Inspector (KHI) timelines, logs, and resource revisions. Use when KHI is running and the user asks to troubleshoot a cluster issue or analyze cluster history.
---

# Investigating Kubernetes Clusters with KHI

## Investigation Workflow

1. Read `khi://inspections` to find an inspection covering the target cluster and time range with `Status` set to `DONE`.
   - If no matching inspection exists, create one following [Creating an Inspection](#creating-an-inspection).
   - If the target inspection has `Status: RUNNING`, call `wait_inspection` until it finishes.
2. Read `khi://inspections/{id}` to inspect the summary (core labels, common properties, per-feature executed queries and log counts, key properties, task reports, and insights) and the `Filter vocabulary` section (`path` keys and example timeline names by `timelineType`, `logType` values, and `severity` constants).
3. Narrow down affected resources and time windows with `search_timelines` and `search_logs` using values from `Filter vocabulary`:
   - Call `search_timelines` to inspect matching timeline nodes with event/revision counts, per-severity log counts including descendants, first/last match timestamps, and compressed revision state transitions (`states`).
   - Call `search_logs` with symptom keywords or severity filters to inspect severity counts, top linked timelines, and evenly spaced sample logs across all timelines.
4. Drill down into specific timelines, logs, and resource state changes:
   - Call `get_timeline_logs` to list logs on a timeline (oldest first) and see other timelines linked to those logs.
   - Call `get_log` to read a log entry's metadata, linked timelines, and full YAML body.
   - Call `get_resource_revisions` to list a timeline's revisions (index, timestamp, verb, state, principal, source log ID, and line diff counts), `get_resource_manifest` to read the resource YAML manifest at a `revisionIndex` or point in `time`, and `get_resource_diff` to inspect the unified diff against the preceding revision.
5. Optionally call `render_timelines` with `filter` and/or `timelineIds` to render the timeline view as a PNG image with a short-lived HTTP image URL.
6. Validate the report and share the Web UI filter:
   - Before presenting the final Markdown report, call `validate_report` to verify that inline code spans (`` `...` ``) containing IDs, resource names, or values exist in the inspection logs. If `validate_report` reports findings, check each flagged item with `search_timelines`, `search_logs`, or `get_log`, and correct or remove values that do not exist in the inspection.
   - Include the `Applied filter` YAML block from the tool responses in the final answer so the user can reproduce the same view in the KHI Web UI.

## Creating an Inspection

1. Read `khi://inspection-types` to find the target `inspectionTypeId`, then call `create_inspection` to create an inspection session and list selectable features.
2. Call `update_inspection_features` with the complete list of `enabledFeatureIds` to enable.
3. If a form field has `Type: file` (such as uploading log files), call `request_file_upload` with `fieldId` to obtain a short-lived single-use `PUT` URL, upload the file with `curl --fail-with-body -X PUT --data-binary @<file> '<URL>'`, and verify `Upload: COMPLETED` in the next `dry_run_inspection` response.
4. Repeat `dry_run_inspection` with `parameters` (strings for text fields, string arrays for set fields, booleans for checkbox fields; all times in UTC RFC3339 format) until it reports `Errors: 0`. Pass `fieldId` to expand truncated suggestions or options for a field. Even when it reports `Errors: 0`, check any `Warning` or `Info` hints on the fields and follow any setup or authentication instructions before calling `run_inspection`.
5. Call `run_inspection` with the validated `parameters`, then repeat `wait_inspection` until the status is no longer `RUNNING`. Call `cancel_inspection` if a running inspection must be aborted.

## Writing Filters

- Read `references/cel-timeline.md` before writing `timelineQuery` or `timelineExclusionQuery`.
- Read `references/cel-log.md` before writing `logQuery`.
- Use only the `path` keys, `timelineType` values, and `logType` values listed in the `Filter vocabulary` section of `khi://inspections/{id}`, `search_timelines`, or `search_logs`.
- KHI reassigns `severity` (`UNKNOWN`, `INFO`, `WARNING`, `ERROR`, `FATAL`) per `logType` for investigation importance; use `body("severity", "ERROR")` to match the raw Cloud Logging severity field.

## Reading More (Pagination and Truncation)

- `pageToken: "..."`: When the last line of `get_timeline_logs` or `get_resource_revisions` is `pageToken: "..."`, pass that `pageToken` to the same tool to read the next page.
- `byteOffset: N`: When the last line of `get_log`, `get_resource_manifest`, or `get_resource_diff` is `byteOffset: N`, pass that `byteOffset` to the same tool to read the rest of the truncated body or diff. For `get_resource_manifest`, always pass `byteOffset` together with the `revisionIndex` shown in the response.

## Handling Errors

The first line of a tool error is `Error: <CODE>`.

- `Error: WORKBENCH_LOADING`: The inspection is still being loaded or indexed into memory. Wait for the duration stated in the error and call the same tool again.
- `Error: INVALID_CEL`: Fix the CEL expression using the error message, `"Did you mean"` suggestions, and the available keys in `Filter vocabulary`, then call the tool again.
- `Error: GOOGLE_CLOUD_AUTH`: Application Default Credentials are missing or expired. Ask the user to run the command shown in the error (`gcloud auth application-default login`) in their local terminal, then retry the tool named in the error.
