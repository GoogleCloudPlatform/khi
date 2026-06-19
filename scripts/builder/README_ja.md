# KHI CI ビルダーイメージ

このディレクトリには、`.cloudbuild/` ワークフローで使用される統合 CI ビルダーイメージ用の Dockerfile が含まれています。

## 含まれる依存関係

- Debian Trixie (ベース OS)
- Go 1.25.x (`golang:1.25-trixie` からコピー)
- Node.js 22.x, npm, npx (`node:22-trixie` のベース環境)
- システムユーティリティ: `jq`, `make`, `git`, `curl`

## ビルドとプッシュ

プロジェクトの `Makefile` を使用してビルドおよびプッシュが行えます。

```bash
# イメージをローカルでビルドする
make build-builder

# イメージをビルドして GCR へプッシュする
make push-builder
```
