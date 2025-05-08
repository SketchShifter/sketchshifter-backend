FROM alpine:3.16

# 必要なパッケージをインストール
RUN apk --no-cache add ca-certificates tzdata && \
    update-ca-certificates

# タイムゾーンを設定
ENV TZ=Asia/Tokyo

# アプリケーションディレクトリを作成
WORKDIR /app

# ビルド済みバイナリをコピー
COPY app /app/app

# アップロードディレクトリを作成して権限を設定
RUN mkdir -p /app/uploads && \
    chmod -R 755 /app/uploads

# 環境変数を設定
ENV SERVER_PORT=8080 \
    UPLOAD_DIR=/app/uploads \
    GIN_MODE=release

# ポートを公開
EXPOSE 8080

# 非rootユーザーに切り替え
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser

# アプリケーションを実行
CMD ["/app/app"]