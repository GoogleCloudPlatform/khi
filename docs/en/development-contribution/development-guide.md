Language: English | [日本語](/docs/ja/development-contribution/development-guide.md)

# Development Guide

This document outlines the steps to set up your development environment to contribute to KHI's code development.
Read [Contribution Guide](contributing.md) and then follow this guide to set up your development environment.

## Run your first build

Follow [the "Run from source code" section](/README.md#run-from-source-code) on README.

## Setup environment for development

### Fork KHI repository

You can't create a new branch our repository directly. Please fork our repository on your account to modify.

### Setup commit signature verification

Please check [this document](https://docs.github.com/en/authentication/managing-commit-signature-verification) to make sure your commits are signed.
Our repository can't accept unsigned commits.

### Setup Git hook

Run the following shell command to setup Git hook. It runs format or lint codes before commiting changes.

```shell
make setup-hooks
```

### Setup VSCode config

Save the following code as `.vscode/launch.json`.

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Start KHI Backend",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "./cmd/kubernetes-history-inspector/",
            "cwd": "${workspaceFolder}",
            "args": [
                "--host",
                "127.0.0.1",
                "--port",
                "8080",
                "--frontend-asset-folder",
                "./dist",
            ],
            "dlvLoadConfig": {
                "followPointers": true,
                "maxVariableRecurse": 1,
                "maxStringLen": 100000,
                "maxArrayValues": 64,
                "maxStructFields": -1
            },
        }
    ], 
}
```

You can run the server with VSCode. You can refer [this document](https://code.visualstudio.com/docs/languages/go) for more details.

### Run frontend server for development

To develop frontend code, we usually start Angular dev server on port 4200 with the following code.

```shell
make watch-web
```

Angular development server on KHI proxies requests to `localhost:4200/api` to `localhost:8080`. ([the proxy config](../../web/proxy.conf.mjs))
You can use KHI with accessing `localhost:4200` instead of `localhost:8080`. Angular dev server automatically build and serve the new build when you change the frontend code.

### Run test

Run the following code to verify frontend and backend codes.

```shell
make test
```

When you want to run backend tests without Cloud Logging, run the following code.

```shell
go test ./... -args -skip-cloud-logging=true
```

## Auto generated codes

### Generated codes from backend codes

Several frontend codes are automativally generated from backend codes.

* `/web/src/app/generated.sass`
* `/web/src/app/generated.ts`

These files are generated with [`scripts/frontend-codegen/main.go` Golang codes](/scripts/frontend-codegen/main.go). It reads several Golang constant arrays and generate frontend codes with templates.

## Markdown Linting

We use markdownlint-cli2 to enforce our documentation style and ensure consistency across our Markdown files.

### Using markdownlint-cli2

The project already includes markdownlint-cli2 as a dev dependency, so you just need to install dependencies:

```bash
npm install
```

To lint Markdown files, run:

```bash
make lint-markdown
```

To automatically fix markdownlint issues:

```bash
make lint-markdown-fix
```

## Releasing container image

KHI automates the container image deployment process.
After creating a dedicated tag by creating a release on GitHub, the container will be built automatically and pushed on the repository.
These tag creations are restricted only for our repository admins.

* Pre-release
  * Name tag with `vx.y.z-beta` then it will be deployed at the following addresses:
    * `asia.gcr.io/kubernetes-history-inspector/release:beta`
    * `asia.gcr.io/kubernetes-history-inspector/release:vx.y.z-beta`
* Release
  * Name tag with `vx.y.z` then it will be deployed at the following address:
    * `asia.gcr.io/kubernetes-history-inspector/release:vx.y.z`
    * `asia.gcr.io/kubernetes-history-inspector/release:latest`

> [!NOTE]
> The deployment process begins after the release entry being created. It may take an hour to push the image on the repository.

### Using on-demand build for pull request code

Repository admins can run `github-deploy-ondemand` check on a Pull request.
It will deploy the image on `asia.gcr.io/kubernetes-history-inspector/develop:$SHORT_SHA`.

> [!NOTE]
> The image is only for the last check. Please check the code is right on your environment first.
> A build may take an hour.
