# OCX Plugin for Shumoku

Open Cloud Exchange (OCX) トポロジープラグイン for Shumoku Server.

## インストール

### 方法1: URLからインストール (推奨)

Shumoku Web UI の Plugins ページで:

1. 「Install from URL」をクリック
2. URL に以下を入力:
   ```
   https://github.com/janog57-noc/network-topology/archive/refs/heads/main.zip
   ```
3. Subdirectory に `shumoku-plugin` を入力
4. Install をクリック

### 方法2: 手動インストール

```bash
# プラグインディレクトリにコピー
cp -r shumoku-plugin $SHUMOKU_DATA_DIR/plugins/ocx

# または plugins.yaml で設定
cat >> $SHUMOKU_DATA_DIR/plugins.yaml << EOF
plugins:
  - id: ocx
    path: /path/to/network-topology/shumoku-plugin
    enabled: true
EOF
```

## 設定

Data Sources ページで新しいデータソースを追加:

| 項目 | 説明 |
|------|------|
| Type | `ocx` (Open Cloud Exchange) |
| Name | 任意の名前 |
| OCX API URL | OCX APIのベースURL |
| Bearer Token | 認証用トークン |
| X-Token-Id | トークンID |

## NetBoxとのマージ

OCX と NetBox のトポロジーをマージするには:

1. **Data Sources** ページで両方のソースを追加
2. **Merge Configuration** ページ (`/topologies/[id]/merge`) で設定:
   - **Base Source**: NetBox を選択
   - **OCX の Match Strategy**: `Manual Mapping` を選択
   - **ID Mapping** に OCX→NetBox のノードIDマッピングを入力:
     ```json
     {
       "135": "PP-01",
       "1686": "PP-02"
     }
     ```
   - **Unmatched Nodes**: `Add to Subgraph`
   - **Subgraph Name**: `ocx`
3. **Save Changes**
4. **Data Sources** ページで **Sync All** をクリック

## ファイル構成

```
shumoku-plugin/
├── plugin.json    # マニフェスト (configSchema含む)
├── index.js       # エントリポイント (register関数)
├── plugin.js      # OCXPlugin クラス
├── client.js      # OCX API クライアント
└── converter.js   # NetworkGraph変換ロジック
```

## 開発

既存の CLI ツールと同じ OCX クライアント/変換ロジックを使用しています:
- `client.js`: `src/ocx-integration/ocx-client.js` から流用
- `converter.js`: `src/ocx-integration/convert.js` から流用
