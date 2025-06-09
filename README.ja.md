<p style="text-align: center;">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./docs/images/logo-dark.svg">
    <img alt="Kubernetes History Inspector" src="./docs/images/logo-light.svg" width="50%">
  </picture>
</p>

Language: [English](./README.md) | 日本語

<hr/>

# Kubernetes History Inspector

Kubernetes History Inspector (KHI) は、Kubernetes クラスタのログ可視化ツールです。
大量のログをインタラクティブなタイムラインビューなどで可視化し、Kubernetes クラスタ内の複数のコンポーネントにまたがる複雑な問題のトラブルシューティングを強力にサポートします。

クラスタ内へのエージェント等のインストールの必要はなく、ログを読み込ませるだけで、トラブルシューティングに役立つ以下のログの可視化を提供します。

|タイムラインビュー|クラスタダイアグラム|
|---|---|
|![Timeline view](./docs/images/timeline.png)|![Cluster diagram](./docs/images/cluster-diagram.png)|
|監査ログ等から特定期間の複数リソースに対する変更、ステータス等の遷移をわかりやすくタイムライン、差分として表示。|kube-apiserverの監査ログから復元した特定タイミングのリソースの関係性をわかりやすく可視化。|

## KHIの特徴

### ログの可視化

KHIの主要な強みは、従来のテキストベースのログ分析を超えて、各Kubernetesリソースに関連する多数のアクティビティログをタイムラインベースのグラフとして視覚化できる点です。
単一のリソースでログを手動でフィルタリングしたり、個々のアクティビティログをテキストデータで時系列に読み進めたりする必要はありません。KHIを使用すると、タイムラインの視覚化から何が起こったのかを一目で把握できます。

また、ログの視覚化に加えて、KHIでは特定の瞬間のログデータを従来のテキスト形式で確認したり、特定のイベント発生時のYAMLマニフェストの差分を確認したりことも可能です。これにより、事象の原因を特定するプロセスが大幅に簡素化されます。

さらに、KHIはある特定の時点でのKubernetesクラスターのリソースの状態とその関係を示すクラスタダイアグラムを生成することもできます。これは、インシデント発生時の特定の時間におけるリソースのステータスやクラスターのトポロジーを理解する上で非常に役に立ちます。

### エージェントレス

KHIのセットアップはとても簡単です。エージェントレスなので、対象クラスターに複雑な事前設定をすることなく、誰でも簡単に使い始めることができます。また、KHIはGUI操作でKubernetesログを視覚化できます。ログの取得のために複雑なクエリやコマンドを記述する必要はありません。
![機能: ログ収集のための迅速かつ簡単なステップ](./docs/ja/images/feature-query.png)

### トラブルシューティングの知見

KHIは、Google Cloud サポートチームが開発し、その後オープンソース化されました。Google Cloudのサポートエンジニアが日々の業務でKubernetesログを分析する中で培った経験から生まれたツールです。KHIには、Kubernetesのログトラブルシューティングにおける彼らの深い専門知識が凝縮されています。

## サポートされている製品

### Kubernetes クラスタ

- Google Cloud

  - [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/docs/concepts/kubernetes-engine-overview)
  - [Cloud Composer](https://cloud.google.com/composer/docs/composer-3/composer-overview)
  - [GKE on AWS](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/aws/concepts/architecture)
  - [GKE on Azure](https://cloud.google.com/kubernetes-engine/multi-cloud/docs/azure/concepts/architecture)
  - [GDCV for Baremetal](https://cloud.google.com/kubernetes-engine/distributed-cloud/bare-metal/docs/concepts/about-bare-metal)
  - [GDCV for VMWare](https://cloud.google.com/kubernetes-engine/distributed-cloud/vmware/docs/overview)

- その他環境
  - JSONlines 形式の kube-apiserver 監査ログ ([チュートリアル (Using KHI with OSS Kubernetes Clusters - Example with Loki | 英語のみ)](/docs/en/setup-guide/oss-kubernetes-clusters.md))

### ログバックエンド

- Google Cloud

  - Cloud Logging（Google Cloud 上のすべてのクラスタ）

- その他環境
  - ファイルによるログアップロード([チュートリアル (Using KHI with OSS Kubernetes Clusters - Example with Loki | 英語のみ)](/docs/en/setup-guide/oss-kubernetes-clusters.md))

## 実行方法

### Docker イメージから実行

#### 動作環境

- Google Chrome（最新版）
- `docker` コマンド

> [!IMPORTANT]
> 動作環境以外でのご利用、または動作環境下でもブラウザの設定によっては正しく動作しない場合がございます。

#### KHI の実行

1. [Cloud Shell](https://shell.cloud.google.com) を開きます。
2. `docker run -p 127.0.0.1:8080:8080 asia.gcr.io/kubernetes-history-inspector/release:latest` を実行します。
3. ターミナル上のリンク `http://localhost:8080` をクリックして、KHI の使用を開始してください！

> [!TIP]
> メタデータサーバーが利用できない他の環境で KHI を実行する場合は、プログラム引数でアクセストークンを渡します。
>
> ```bash
> docker run -p 127.0.0.1:8080:8080 asia.gcr.io/kubernetes-history-inspector/release:latest -access-token=`gcloud auth print-access-token`
> ```

> [!NOTE]
> コンテナイメージの配信元は近いうちに変更される可能性があります。 #21

詳細は [Getting Started](/docs/en/tutorial/getting-started.md) を参照してください。

### ソースから実行

<details>
<summary>動かしてみる (ソースから実行)</summary>

#### ビルドに必要な依存関係

- Go 1.24.\*
- Node.js 環境 22.13.\*
- [`gcloud` CLI](https://cloud.google.com/sdk/docs/install)
- [`jq`コマンド](https://jqlang.org/)

#### 環境構築

1. このリポジトリをダウンロードまたはクローンします。  
   例: `git clone https://github.com/GoogleCloudPlatform/khi.git`
2. プロジェクトルートに移動します。  
   例: `cd khi`
3. プロジェクトルートから `cd ./web && npm install` を実行します。

#### KHI のビルドと実行

1. [`gcloud` で認証します。](https://cloud.google.com/docs/authentication/gcloud)  
   例: ユーザーアカウントの認証情報を使用する場合は、`gcloud auth login` を実行します。
2. プロジェクトルートから `make build-web && KHI_FRONTEND_ASSET_FOLDER=./dist go run cmd/kubernetes-history-inspector/main.go` を実行します。  
   `localhost:8080` を開き、KHI の使用を開始してください！

</details>

> [!IMPORTANT]
> KHI のポートをインターネット向けに公開しないでください。
> KHI 自身は認証、認可の機能を提供しておらず、ローカルユーザからのみアクセスされることが想定されています。

### 権限設定

## マネージド環境毎の設定

### Google Cloud

#### 権限

以下の権限が必須・推奨されます。

- **必須権限**
  - `logging.logEntries.list`
- **推奨権限**
  - 対象のクラスタのタイプに対するリスト権限（例：GKE の場合 `container.clusters.list`）
    ログフィルタ生成ダイアログの候補の出力に使用します。KHI の主機能の利用に影響はありません。
- **設定手順**

  - Compute Engine 仮想マシン上など、サービスアカウントがアタッチされた Google Cloud 環境で KHI を実行する場合、対応するリソースにアタッチされたサービスアカウントに上記権限を付与します。
  - ローカル環境や Cloud Shell など、ユーザアカウント権限で KHI を実行する場合、対応するユーザ上記権限を付与します。

> [!WARNING]
> KHI は、Compute Engine インスタンス上で実行した際は必ずアタッチされたサービスアカウントを使用するなど、[ADC](https://cloud.google.com/docs/authentication/provide-credentials-adc)が反映されません。
> この仕様は今後修正される場合があります。

#### 監査ログ出力設定

- **必須設定無し**
- **推奨設定**
  - Kubernetes Engine API データ書き込み監査ログの有効化

> [!TIP]
> Pod や Node リソースの`.status`フィールドへのパッチリクエストが記録されており、
> トラブルシューティングに詳細なコンテナの情報も必要な場合に推奨されます。
> Kubernetes Engine API データ書き込み監査ログが未出力の場合も、KHI は Pod 削除時の監査ログから最終のコンテナの状態を表示できますが、Pod が削除されない間のコンテナの状態変化が記録されません。

- **設定手順**
  1. Google Cloud コンソールで、[監査ログページに移動](https://console.cloud.google.com/iam-admin/audit)します。
  1. 「データアクセス監査ログの構成」以下の、「サービス」列から「Kubernetes Engine API」を選択します。
  1. 「ログタイプ」タブで、「データ書き込み」を選択します。
  1. 「保存」をクリックします。

### OSS Kubernetes

[OSS Kubernetesクラスタのログの可視化（Loki）](/docs/ja/setup-guide/oss-kubernetes-clusters.md)を参照してください。

## ユーザーガイド

[ユーザーガイド](/docs/ja/visualization-guide/user-guide.md) をご確認ください。

## KHIプロジェクトへの貢献

プロジェクトへの貢献をご希望の場合は、[コントリビューションガイド](/docs/en/development-contribution/contributing.md) をお読みの上、[KHI開発環境のセットアップ](/docs/ja/development-contribution/development-guide.md)を実施してください。

## 免責事項

KHI は Google Cloud の公式製品ではございません。不具合のご報告や機能に関するご要望がございましたら、お手数ですが当リポジトリの[Github issues](https://github.com/GoogleCloudPlatform/khi/issues/new?template=Blank+issue)にご登録ください。可能な範囲で対応させていただきます。
