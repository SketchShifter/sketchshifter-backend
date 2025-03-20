# SketchShifter Backend

[swaggerはこちらです](https://mmm-tapj.vercel.app/)


### とりあえずインストール

```bash
# リポジトリのクローン
git clone https://github.com/SketchShifter/sketchshifter_backend.git
cd github.com/SketchShifter/sketchshifter_backend/

# 依存パッケージのインストール
go mod tidy
go mod download

# ビルド
make build
make run
make migrate-up
```

Processingで作成した作品を共有・閲覧するためのプラットフォームのバックエンドAPIです。

## 機能

- RESTful API提供
- ユーザー認証（JWT）
- 作品のアップロード・管理
- コメント・いいね・お気に入り機能
- タグ管理
- ゲスト投稿機能

## 環境構築

### 必要条件

- Go 1.21以上
- PostgreSQL 14以上
- Docker & Docker Compose (任意)


### Dockerでの起動

```bash
# Dockerイメージビルド＆コンテナ起動
docker-compose up -d

# ログ確認
docker-compose logs -f
```

## API仕様

### 認証

- `POST /api/auth/register` - ユーザー登録
- `POST /api/auth/login` - ログイン
- `GET /api/auth/me` - 現在のユーザー情報

### 作品

- `GET /api/works` - 作品一覧取得
- `GET /api/works/:id` - 作品詳細取得
- `POST /api/works` - 作品投稿
- `PUT /api/works/:id` - 作品更新
- `DELETE /api/works/:id` - 作品削除
- `POST /api/works/preview` - プレビュー

### いいね/お気に入り

- `POST /api/works/:id/like` - いいね追加
- `DELETE /api/works/:id/like` - いいね削除
- `POST /api/works/:id/favorite` - お気に入り追加
- `DELETE /api/works/:id/favorite` - お気に入り削除

### コメント

- `GET /api/works/:id/comments` - コメント取得
- `POST /api/works/:id/comments` - コメント追加
- `PUT /api/comments/:id` - コメント更新
- `DELETE /api/comments/:id` - コメント削除

### タグ

- `GET /api/tags` - タグ一覧取得

### ユーザー

- `GET /api/users/favorites` - お気に入り作品取得
- `GET /api/users/:id` - ユーザー情報取得
- `GET /api/users/:id/works` - ユーザー作品一覧取得

## 開発

このプロジェクトは現在モックデータを返すようになっています。実際のデータベース連携やストレージ実装は別途必要です！
