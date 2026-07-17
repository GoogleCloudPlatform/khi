Language: [English](/docs/en/setup-guide/google-cloud-permissions.md) | 日本語

# Google Cloud 権限設定および認証・監査ログガイド

Kubernetes History Inspector (KHI) は、Google Cloud Logging から Kubernetes 監査ログを取得します。GKE クラスタや Google Cloud 環境に対するインスペクション（解析）を実行するには、KHI を実行するアカウントに適切な IAM 権限と監査ログの設定が必要です。

## IAM 権限設定

### 必須・推奨権限

KHI が Cloud Logging をクエリし、New Inspection ダイアログでの入力補完候補を取得するために以下の権限が必要です。

* **必須権限**:
  * `logging.logEntries.list` - Cloud Logging からログエントリを取得するために使用します。
* **推奨権限**:
  * `monitoring.timeSeries.list` - New Inspection ダイアログでクラスタ名などの自動補完候補を取得するために使用します（権限がなくても機能しますが、候補が表示されません）。
  * `container.clusters.list` - Cloud Composer 向け機能利用時のクラスタメタデータ取得に使用します。

### おすすめの IAM ロール

個別に権限を設定する代わりに、以下の標準 IAM ロールのいずれかをユーザーまたはサービスアカウントに付与できます。

| IAM ロール | ロール ID | 目的 |
| --- | --- | --- |
| **ログ閲覧者 (Logs Viewer)** | `roles/logging.viewer` | Cloud Logging の標準的なログのクエリを許可します。 |
| **プライベート ログ閲覧者 (Private Logs Viewer)** | `roles/logging.privateLogViewer` | センシティブなデータを含む監査ログのクエリを許可します。（完全な監査ログアクセスのため推奨） |

---

## 認証方法

KHI は Google Cloud への API リクエストの認証に Application Default Credentials (ADC) を使用します。

### 1. Cloud Shell で実行する場合（設定不要）

Google Cloud Shell 上で KHI を実行する場合、Cloud Shell のメタデータ サービスが自動的に使用されます。認証情報ファイルの追加マウントは不要です。

```bash
docker run -p 127.0.0.1:8080:8080 gcr.io/kubernetes-history-inspector/release:latest
```

### 2. ローカル環境で実行する場合（gcloud ADC）

ローカルマシン（Linux, macOS, Windows）上で KHI を実行する場合、`gcloud` コマンドで ADC を生成し、そのファイルをコンテナ内にマウントします。

1. アプリケーションデフォルト認証情報を生成します:

   ```bash
   gcloud auth application-default login
   ```

2. 認証情報ファイルをマウントして KHI コンテナを実行します:

   **Linux / macOS / WSL 環境:**

   ```bash
   docker run \
     -p 127.0.0.1:8080:8080 \
     -v ~/.config/gcloud/application_default_credentials.json:/root/.config/gcloud/application_default_credentials.json:ro \
     gcr.io/kubernetes-history-inspector/release:latest
   ```

   **Windows PowerShell 環境:**

   ```bash
   docker run `
     -p 127.0.0.1:8080:8080 `
     -v $env:APPDATA\gcloud\application_default_credentials.json:/root/.config/gcloud/application_default_credentials.json:ro `
     gcr.io/kubernetes-history-inspector/release:latest
   ```

### 3. サービスアカウントキーを使用する場合

Compute Engine 仮想マシン等の環境や専用サービスアカウントで使用する場合は、アタッチされたサービスアカウントに上記権限を付与するか、キーファイルをマウントして `GOOGLE_APPLICATION_CREDENTIALS` を設定します:

```bash
docker run \
  -p 127.0.0.1:8080:8080 \
  -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/sa-key.json \
  -v /path/to/sa-key.json:/tmp/sa-key.json:ro \
  gcr.io/kubernetes-history-inspector/release:latest
```

### 4. サービスアカウントのなりすまし（Impersonation）

サービスアカウントになりすまして実行する場合:

```bash
gcloud auth application-default login --impersonate-service-account=<SERVICE_ACCOUNT_EMAIL>
```

---

## 監査ログ出力設定

### Kubernetes Engine API 監査ログ

* **デフォルト**: KHI はデフォルトの Google Cloud 監査ログ構成で問題なく動作します。
* **推奨設定**: Kubernetes Engine API の「データ書き込み (DATA_WRITE)」データアクセス監査ログの有効化。

> [!TIP]
> 「データ書き込み」監査ログを有効にすると、Pod や Node リソースの `.status` フィールドへのパッチリクエストが記録され、コンテナの状態遷移をより詳細に可視化できます。無効の場合でも KHI は Pod 削除ログから最終状態を推測できますが、削除前の状態変化の記録精度が向上します。

### 設定手順

1. Google Cloud コンソールの [監査ログページ](https://console.cloud.google.com/iam-admin/audit) に移動します。
2. データアクセス監査ログ構成テーブルの「サービス」列から **Kubernetes Engine API** を選択します。
3. 「ログタイプ」タブで **データ書き込み (Data write)** を選択します。
4. **保存** をクリックします。
