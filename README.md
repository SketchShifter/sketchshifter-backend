# Processing作品共有プラットフォーム

## セットアップ方法

これだけは見て！！！！！！！！
```bash
# ローカル環境で実行
./local-setup.sh

# サーバー環境で実行
./setup-script-specific.sh

# 本番モードで実行
make prod

# マイグレーション
make migrate-up

# 停止
make stop

# ログ表示
make logs
```

### 必要条件

- Docker
- Docker Compose
- make (オプション、便利なコマンドが使えます)

### インストールと実行方法

1. 開発モードで実行:

```bash
make run
```

2. 本番モードで実行:

```bash
make prod
```

3. マイグレーションを実行:

```bash
make migrate-up
```

4. ログを表示:

```bash
make logs      # 開発モード
make prod-logs # 本番モード
```

### アクセス方法

- API: http://localhost:8080
- PHPMyAdmin: http://localhost:8081 (開発モードのみ)

### データベース情報

- ホスト: localhost
- ポート: 3306
- ユーザー名: processing_user
- パスワード: processing_password
- データベース名: processing_platform

## ディレクトリ構造

```
.
├── cmd/                  # エントリーポイント
│   ├── app/              # メインアプリケーション
│   └── migrate/          # マイグレーションツール
├── internal/             # 内部パッケージ
│   ├── config/           # 設定
│   ├── controllers/      # コントローラー
│   ├── middlewares/      # ミドルウェア
│   ├── models/           # データモデル
│   ├── repository/       # データアクセス層
│   ├── routes/           # ルーティング
│   ├── services/         # ビジネスロジック
│   └── utils/            # ユーティリティ
├── pkg/                  # 外部パッケージ
├── uploads/              # アップロードされたファイル
├── Dockerfile            # 開発用Dockerfile
├── Dockerfile.prod       # 本番用Dockerfile
├── docker-compose.yml    # 開発用Docker Compose設定
├── docker-compose.prod.yml # 本番用Docker Compose設定
├── go.mod                # Goモジュール定義
└── Makefile              # ビルド・実行タスク
```
