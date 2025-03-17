# sketchshifter_backend(バックエンド)
```bash
# リポジトリのクローン
git clone https://github.com/SketchShifter/sketchshifter_backend.git
cd SketchShifter/sketchshifter_backend.git

# 依存パッケージのインストール
go mod tidy

# 開発用サーバー起動
go run cmd/main.go
```

Go言語とGinフレームワークを使用したProcessing作品共有プラットフォームのバックエンド実装です。

## 機能概要

- RESTful API提供
- JWT認証
- ファイルアップロード処理
- データベース操作（PostgreSQL）
- Processing作品の管理

## 環境構築

### 必要条件

- Go 1.20以上
- PostgreSQL 14以上
- (任意) Docker & Docker Compose

### インストール

```bash
# リポジトリのクローン
git clone https://github.com/SketchShifter/sketchshifter_backend.git
cd SketchShifter/sketchshifter_backend.git

# 依存パッケージのインストール
go mod tidy

# 開発用サーバー起動
go run cmd/main.go
```

### Dockerでの起動

```bash
# Dockerイメージビルド＆コンテナ起動
docker-compose up -d

# ログ確認
docker-compose logs -f
```

### 環境変数の設定

`.env`ファイルをプロジェクトルートに作成し、以下の変数を設定します：

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=processingshare
JWT_SECRET=your_jwt_secret_key
STORAGE_PATH=./uploads
PORT=8080
```

## ディレクトリ構成

```
.
├── cmd/                      # エントリーポイント
│   └── main.go               # メインアプリケーション
├── config/                   # 設定ファイル
│   └── config.go             # 環境設定
├── controllers/              # コントローラー
│   ├── auth.go               # 認証
│   ├── works.go              # 作品
│   ├── comments.go           # コメント
│   └── users.go              # ユーザー
├── models/                   # データモデル
│   ├── user.go               # ユーザーモデル
│   ├── work.go               # 作品モデル
│   └── comment.go            # コメントモデル
├── repository/               # データアクセス層
│   └── postgres/             # PostgreSQL実装
├── middlewares/              # ミドルウェア
│   ├── auth.go               # 認証ミドルウェア
│   └── cors.go               # CORSミドルウェア
├── routes/                   # ルーティング
│   └── routes.go             # APIルート定義
├── services/                 # ビジネスロジック
│   ├── auth.go               # 認証サービス
│   └── storage.go            # ファイルストレージ
├── utils/                    # ユーティリティ
│   ├── jwt.go                # JWT処理
│   └── validator.go          # バリデーション
├── migrations/               # DBマイグレーション
├── .env                      # 環境変数
├── docker-compose.yml        # Docker設定
├── Dockerfile                # Dockerイメージ
└── go.mod                    # Goモジュール
```

## API仕様

主なエンドポイント：

- **認証**
  - `POST /api/auth/register` - ユーザー登録
  - `POST /api/auth/login` - ログイン
  - `GET /api/auth/me` - 現在のユーザー情報

- **作品**
  - `GET /api/works` - 作品一覧取得
  - `GET /api/works/:id` - 作品詳細取得
  - `POST /api/works` - 作品投稿
  - `PUT /api/works/:id` - 作品更新
  - `DELETE /api/works/:id` - 作品削除
  - `POST /api/works/preview` - プレビュー

- **いいね/お気に入り**
  - `POST /api/works/:id/like` - いいね追加
  - `DELETE /api/works/:id/like` - いいね削除
  - `POST /api/works/:id/favorite` - お気に入り追加
  - `DELETE /api/works/:id/favorite` - お気に入り削除

- **コメント**
  - `GET /api/works/:id/comments` - コメント取得
  - `POST /api/works/:id/comments` - コメント追加
  - `PUT /api/comments/:id` - コメント更新
  - `DELETE /api/comments/:id` - コメント削除

## データベースマイグレーション

```bash
# マイグレーション実行
go run cmd/migrate/main.go up

# マイグレーションロールバック
go run cmd/migrate/main.go down
```


## 本番デプロイ

```bash
# バイナリビルド
go build -o app cmd/main.go

# 実行
./app
```
