Language: [English](/docs/en/development-contribution/development-guide.md) | 日本語

# KHI開発環境のセットアップ

本ドキュメントは、KHIのコード開発に貢献するために開発環境をセットアップする手順を記載しています。
まずは [Contribution Guide](/docs/en/development-contribution/contributing.md) をお読みいただいた上で、本ドキュメントに沿って開発環境をセットアップしてください。

## KHIをビルドする

READMEの[ソースから実行](/README.ja.md#ソースから実行) の手順に従ってください。

## 開発環境のセットアップ

### KHIレポジトリをforkする

[KHIレポジトリ](https://github.com/GoogleCloudPlatform/khi)に直接新しいブランチを作成することはできません。あなたのアカウントにKHIレポジトリをforkしてください。

### コミット署名の設定

[こちらのドキュメント](https://docs.github.com/en/authentication/managing-commit-signature-verification) の手順に沿って、コミットに署名を付与するように設定してください。コミット署名なしのコミットは受付できません。

### Git hookの設定

下記のシェルコマンドを実行してGit hookを設定してください。下記の設定が完了すると、コミットする前にフォーマットやリントをチェックするコードが実行されます。

```shell
make setup-hooks
```

### VSCodeの設定

下記のコードを`.vscode/launch.json`ファイルとして保存してください。

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

VSCodeでKHIサーバーを実行できます。詳しくは[こちらのドキュメント](https://code.visualstudio.com/docs/languages/go) をご参照ください。

### フロントエンドサーバーの実行

フロントエンドの開発を実施する際、下記のコードを実行すると開発環境のAngularサーバーを4200番ポートで実行できます。

```shell
make watch-web
```

KHIの開発環境のAngularサーバーはリクエストを `localhost:4200/api` から`localhost:8080`にプロキシします([the proxy config](../../web/proxy.conf.mjs))。
 `localhost:8080`ではなく `localhost:4200` にてKHIにアクセスできます。 開発環境のAngularサーバーは自動的にビルドされ、フロントエンドのコードの変更が自動で適用されます。

### テストの実行

下記を実行すると、フロントエンドとバックエンドのコードのテストが実行されます。

```shell
make test
```

バックエンドのテストをCloud Loggingと一緒に実行したい場合は下記のコードを実行してください。

```shell
go test ./... -args -skip-cloud-logging=true
```

## 自動生成コード

### バックエンドコードから自動作成されるフロントエンドコード

下記のフロントエンドのコードは、バックエンドのコードから自動生成されます。

* `/web/src/app/generated.sass`
* `/web/src/app/generated.ts`

上記のファイルは [`scripts/frontend-codegen/main.go` Golang codes](/scripts/frontend-codegen/main.go)にて、Golang側の一部の定数からテンプレートをもとに生成されます。

## マークダウンリンター

KHIではmarkdownlint-cli2を使用して、Markdownファイルにおけるキュメントのスタイルを構成します。

### markdownlint-cli2の使用

KHIプロジェクトは markdownlint-cli2 をディペンデンシーとして含んでいるため、下記をインストールする必要があります。

```bash
npm install
```

下記のコマンドでリンターが実行されます:

```bash
make lint-markdown
```

マークダウンを自動的に修正するには下記を実行します:

```bash
make lint-markdown-fix
```

## コンテナイメージのリリース

KHIはコンテナイメージのデプロイプロセスを自動化しています。
GitHubでリリースを作成すると、専用のタグが自動的に生成されます。この操作がトリガーとなり、コンテナが自動的にビルドされ、リポジトリにプッシュされます。

* プレリリース
  * tagを `vx.y.z-beta`として命名すると、 下記のアドレスとしてデプロイされます。
    * `asia.gcr.io/kubernetes-history-inspector/release:beta`
    * `asia.gcr.io/kubernetes-history-inspector/release:vx.y.z-beta`
* リリース
  * tagを`vx.y.z` として命名すると、 下記のアドレスとしてデプロイされます。
    * `asia.gcr.io/kubernetes-history-inspector/release:vx.y.z`
    * `asia.gcr.io/kubernetes-history-inspector/release:latest`

> [!NOTE]
> リリースの作成後にデプロイプロセスが開始されます。イメージがリポジトリにプッシュされるまで1時間ほどかかる場合があります。

### プルリクエストのコードに対するオンデマンドビルドの使用

レポジトリ管理者は、プルしクエストに対して `github-deploy-ondemand` チェックを実行できます。これによりイメージが`asia.gcr.io/kubernetes-history-inspector/develop:$SHORT_SHA`にデプロイされます。

> [!NOTE]
> このイメージは、最後のチェックのためだけのものです。まず、あなたの環境でコードが正しいことを確認してください。
ビルドには1時間かかる場合があります。
