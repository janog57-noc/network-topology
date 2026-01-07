# Network Topology Generator

注記: 本リポジトリに含まれる情報はすべて公開情報です。

## 概要

- NetBox から機器・インターフェイス・ケーブル情報を取得し物理トポロジー図を生成
- 出力は `out/` 配下に `topology.dot` / `topology.svg` / `topology.png` / `index.html` として保存
- Netbox Webhook をトリガーとして配線情報に変更があれば GitHub Actions で自動生成・デプロイ
- GitHub Pages で [https://janog57-noc.github.io/network-topology/] に生成

![Sample Output](docs/images//topology.png)

## 必要なもの

- Go 1.21 以降
- Graphviz (`dot` コマンドが必要)
- NetBox API アクセス用の環境変数
  - `NETBOX_URL`
  - `NETBOX_TOKEN`

## 使い方（ローカル）

```bash
make run
```

- 上記で依存取得・ビルド・実行を行い、成果物が `out/` に生成される
- 既存の成果物を消したい場合は `rm -rf out` を実行

### Web サーバーで確認

```bash
python3 -m http.server 8000
```

上記でWebサーバーを起動し、 `http://localhost:8000/out/index.html` を確認

## 主要コマンド

- `make build` : バイナリをビルド
- `make run`   : バイナリをビルドして実行（生成物を out/ に出力）
- `make clean` : バイナリと生成物を削除

## GitHub Actions

- ワークフロー: `.github/workflows/netbox-pages.yml`
- トリガー:  `repository_dispatch` (NetBox Webhook 用), `push` (main), 手動実行
