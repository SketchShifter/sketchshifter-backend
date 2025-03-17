FROM golang:1.21-alpine

WORKDIR /app

# 依存関係をコピー
COPY go.mod go.sum* ./
RUN go mod download

# ソースコードをコピー
COPY . .

# 環境変数の設定
ENV GIN_MODE=release

# ポートを公開
EXPOSE 8080

# アプリケーションを実行
CMD ["go", "run", "cmd/app/main.go"]
