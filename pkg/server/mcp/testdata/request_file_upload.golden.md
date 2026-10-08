# Upload URL for `log-file`

- URL: `http://127.0.0.1:8080/api/v1/file-upload/ABCDEF`
- Method: PUT
- Max size: 1024 bytes
- Expires at: 2026-09-24T02:00:00Z

Send the file body in one request. The URL accepts only one upload; if the upload fails, call `request_file_upload` again for a new URL:

```sh
curl --fail-with-body -X PUT --data-binary @./path/to/file 'http://127.0.0.1:8080/api/v1/file-upload/ABCDEF'
```

Then call `dry_run_inspection` and check that the upload status is COMPLETED.
