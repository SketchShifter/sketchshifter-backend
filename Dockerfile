FROM golang:1.18-alpine

# 依存関係のためのパッケージをインストール
RUN apk add --no-cache git gcc musl-dev

# 作業ディレクトリを設定
WORKDIR /app

# Go Modulesのキャッシュレイヤーを作成
COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy

# ソースコードをコピー
COPY . .

# アップロードディレクトリを作成
RUN mkdir -p /app/uploads

# 環境変数を設定（デフォルト値）
ENV SERVER_PORT=8080 \
    UPLOAD_DIR=/app/uploads \
    GIN_MODE=debug

# ポートを公開
EXPOSE 8080

# スタートアップスクリプトをコピー
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# エントリーポイントを設定
ENTRYPOINT ["docker-entrypoint.sh"]

# デフォルトコマンド
CMD ["go", "run", "./cmd/app/main.go"]