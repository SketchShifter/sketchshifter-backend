#!/bin/bash

# Processing作品共有プラットフォーム EC2セットアップスクリプト
# このスクリプトは、Amazon Linux 2または2023上でProcessing作品共有プラットフォームを
# 指定のディレクトリに設定するためのものです。

set -e

# アプリケーションディレクトリ設定
APP_DIR="/home/ec2-user/ssjs/sketchshifter_backend"

# カラー定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# ヘルパー関数
print_step() {
    echo -e "${GREEN}==>${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}警告:${NC} $1"
}

print_error() {
    echo -e "${RED}エラー:${NC} $1"
}

# バナー表示
echo -e "${GREEN}"
echo "============================================================="
echo "  Processing作品共有プラットフォーム Amazon Linux セットアップ  "
echo "============================================================="
echo -e "${NC}"

# システムチェック
print_step "システム情報を確認しています..."
if grep -q "Amazon Linux" /etc/os-release; then
    echo "Amazon Linux が検出されました"
    # Amazon Linuxのバージョンチェック
    if grep -q "Amazon Linux 2" /etc/os-release; then
        echo "Amazon Linux 2 が検出されましたが、2023に変更します"
        AMAZON_LINUX_VERSION="2023"
    else
        echo "Amazon Linux 2023 が検出されました"
        AMAZON_LINUX_VERSION="2023"
    fi
else
    print_error "このスクリプトはAmazon Linux専用です。他のディストリビューションでは正常に動作しない可能性があります。"
    read -p "続行しますか？ (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 必要なディレクトリの設定
mkdir -p "$APP_DIR"
cd "$APP_DIR"

# システムアップデート
print_step "1. システムを更新しています..."
sudo yum update -y

# Dockerインストール
print_step "2. Dockerをインストールしています..."
if ! command -v docker &> /dev/null; then
    echo "Dockerをインストール中..."
    if [ "$AMAZON_LINUX_VERSION" = "2" ]; then
        sudo amazon-linux-extras install docker -y
    else
        sudo yum install docker -y
    fi
    
    sudo systemctl start docker
    sudo systemctl enable docker
    sudo usermod -a -G docker $USER
    echo "Dockerのインストールが完了しました。変更を適用するにはシェルを再起動する必要があります。"
else
    echo "Dockerはすでにインストールされています"
fi

# Docker Composeインストール
print_step "3. Docker Composeをインストールしています..."
if ! command -v docker-compose &> /dev/null; then
    echo "Docker Composeをインストール中..."
    LATEST_COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    sudo curl -L "https://github.com/docker/compose/releases/download/${LATEST_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    echo "Docker Composeのインストールが完了しました"
else
    echo "Docker Composeはすでにインストールされています"
fi

# Gitインストール
print_step "4. Gitをインストールしています..."
if ! command -v git &> /dev/null; then
    sudo yum install git -y
    echo "Gitのインストールが完了しました"
else
    echo "Gitはすでにインストールされています"
fi

# makeのインストール
print_step "5. makeをインストールしています..."
sudo yum install make -y
echo "makeのインストールが完了しました"

# 設定ファイルの準備
print_step "6. アプリケーションを準備しています..."

# docker-entrypoint.shの作成
if [ ! -f "$APP_DIR/docker-entrypoint.sh" ]; then
    cat > "$APP_DIR/docker-entrypoint.sh" << 'EOF'
#!/bin/sh
set -e

echo "=== Processingプラットフォーム API サーバー ==="
echo "環境変数:"
echo "SERVER_PORT: $SERVER_PORT"
echo "DB_HOST: $DB_HOST"
echo "DB_NAME: $DB_NAME"
echo "UPLOAD_DIR: $UPLOAD_DIR"
echo "GIN_MODE: $GIN_MODE"
echo "========================================"

# 引数をそのまま実行
echo "サーバーを起動します..."
exec "$@"
EOF
    chmod +x "$APP_DIR/docker-entrypoint.sh"
    echo "docker-entrypoint.shを作成しました"
fi

# Dockerfile.prodの作成
if [ ! -f "$APP_DIR/Dockerfile.prod" ]; then
    cat > "$APP_DIR/Dockerfile.prod" << 'EOF'
FROM golang:1.18-alpine AS builder

# 依存関係のためのパッケージをインストール
RUN apk add --no-cache git gcc musl-dev

# 作業ディレクトリを設定
WORKDIR /app

# Go Modulesのキャッシュレイヤーを作成
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ソースコードをコピー
COPY . .

# バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o app ./cmd/app/main.go

# 最終イメージを小さくするためのマルチステージビルド
FROM alpine:3.16

# 必要なパッケージをインストール
RUN apk --no-cache add ca-certificates tzdata && \
    update-ca-certificates

# タイムゾーンを設定
ENV TZ=Asia/Tokyo

# ビルドしたバイナリをコピー
COPY --from=builder /app/app /app/app

# 設定ファイルとスクリプトをコピー
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# アップロードディレクトリを作成
RUN mkdir -p /app/uploads

# 作業ディレクトリを設定
WORKDIR /app

# 環境変数を設定（デフォルト値）
ENV SERVER_PORT=8080 \
    UPLOAD_DIR=/app/uploads \
    GIN_MODE=release

# ポートを公開
EXPOSE 8080

# 非rootユーザーに切り替え
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser

# エントリーポイントを設定
ENTRYPOINT ["docker-entrypoint.sh"]

# アプリケーションを実行
CMD ["/app/app"]
EOF
    echo "Dockerfile.prodを作成しました"
fi

# docker-compose.prod.ymlの作成
if [ ! -f "$APP_DIR/docker-compose.prod.yml" ]; then
    cat > "$APP_DIR/docker-compose.prod.yml" << 'EOF'
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile.prod
    container_name: processing-api
    restart: always
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - DB_HOST=db
      - DB_PORT=3306
      - DB_USER=${DB_USER:-processing_user}
      - DB_PASSWORD=${DB_PASSWORD:-processing_password}
      - DB_NAME=${DB_NAME:-processing_platform}
      - JWT_SECRET=${JWT_SECRET:-your-jwt-secret-key-change-this}
      - TOKEN_EXPIRY=24
      - UPLOAD_DIR=/app/uploads
      - MAX_UPLOAD_SIZE=50
      - GIN_MODE=release
    volumes:
      - ./uploads:/app/uploads
    depends_on:
      - db
    networks:
      - processing_network
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s

  db:
    image: mariadb:10.6
    container_name: processing-db
    restart: always
    ports:
      - "127.0.0.1:3306:3306"
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-root_password}
      - MYSQL_DATABASE=${DB_NAME:-processing_platform}
      - MYSQL_USER=${DB_USER:-processing_user}
      - MYSQL_PASSWORD=${DB_PASSWORD:-processing_password}
    volumes:
      - db_data:/var/lib/mysql
    networks:
      - processing_network
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD:-root_password}"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

networks:
  processing_network:
    driver: bridge

volumes:
  db_data:
    name: processing_db_data
EOF
    echo "docker-compose.prod.ymlを作成しました"
fi

# .envファイルの作成
print_step "6. 環境変数を設定しています..."
if [ ! -f "$APP_DIR/.env" ]; then
    cat > "$APP_DIR/.env" << EOF
DB_USER=processing_user
EOF
    
    # ランダムなパスワードと秘密鍵を生成
    if command -v openssl >/dev/null 2>&1; then
        echo "DB_PASSWORD=$(openssl rand -base64 12 | tr -dc 'a-zA-Z0-9')" >> "$APP_DIR/.env"
        echo "MYSQL_ROOT_PASSWORD=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9')" >> "$APP_DIR/.env"
        echo "JWT_SECRET=$(openssl rand -base64 32)" >> "$APP_DIR/.env"
    else
        echo "DB_PASSWORD=$(< /dev/urandom tr -dc 'a-zA-Z0-9' | fold -w 12 | head -n 1)" >> "$APP_DIR/.env"
        echo "MYSQL_ROOT_PASSWORD=$(< /dev/urandom tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)" >> "$APP_DIR/.env"
        echo "JWT_SECRET=$(< /dev/urandom tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)" >> "$APP_DIR/.env"
    fi
    
    echo "DB_NAME=processing_platform" >> "$APP_DIR/.env"
    echo ".envファイルを作成しました"
fi

# アップロードディレクトリの作成
print_step "7. アップロードディレクトリを作成しています..."
mkdir -p "$APP_DIR/uploads"
chmod 755 "$APP_DIR/uploads"

# システム起動時の自動起動スクリプトの作成
print_step "8. システム起動時の自動起動を設定しています..."
if [ ! -f "/etc/systemd/system/sketchshifter.service" ]; then
    sudo tee /etc/systemd/system/sketchshifter.service > /dev/null << EOF
[Unit]
Description=SketchShifter Processing Platform API
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$APP_DIR
ExecStart=/usr/bin/docker-compose -f docker-compose.prod.yml up -d
ExecStop=/usr/bin/docker-compose -f docker-compose.prod.yml down
User=$USER
Group=$USER
TimeoutStartSec=180
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl daemon-reload
    sudo systemctl enable sketchshifter.service
    echo "自動起動サービスを設定しました"
else
    echo "自動起動サービスはすでに設定されています"
fi

# バックアップスクリプトの作成
print_step "9. バックアップスクリプトを設定しています..."

# crontabコマンドがない場合はインストール
if ! command -v crontab &> /dev/null; then
    echo "crontabコマンドが見つかりません。必要なパッケージをインストールします..."
    
    if [ "$AMAZON_LINUX_VERSION" = "2" ]; then
        # Amazon Linux 2ではcrontabはデフォルトでインストールされているはず
        sudo yum install -y cronie
    else
        # Amazon Linux 2023の場合
        sudo dnf install -y cronie
    fi
    
    # crondサービスを開始・有効化
    sudo systemctl start crond
    sudo systemctl enable crond
    
    echo "cronパッケージをインストールしました"
fi

BACKUP_SCRIPT="$HOME/backup.sh"
if [ ! -f "$BACKUP_SCRIPT" ]; then
    cat > "$BACKUP_SCRIPT" << EOF
#!/bin/bash
# 自動バックアップスクリプト

# バックアップディレクトリの設定
BACKUP_DIR="\$HOME/backups"
TIMESTAMP=\$(date +"%Y%m%d_%H%M%S")
APP_DIR="$APP_DIR"

# ディレクトリが存在しない場合は作成
mkdir -p "\$BACKUP_DIR"

# 環境変数の読み込み
cd "\$APP_DIR"
if [ -f .env ]; then
    source .env
else
    echo "環境変数ファイル(.env)が見つかりません"
    exit 1
fi

# データベースのバックアップ
echo "データベースをバックアップ中..."
DB_BACKUP_FILE="\$BACKUP_DIR/db_backup_\$TIMESTAMP.sql"
docker-compose -f docker-compose.prod.yml exec -T db mysqldump \\
    -u root \\
    -p"\${MYSQL_ROOT_PASSWORD:-root_password}" \\
    "\${DB_NAME:-processing_platform}" > "\$DB_BACKUP_FILE"

# バックアップが正常に作成されたか確認
if [ -s "\$DB_BACKUP_FILE" ]; then
    echo "データベースのバックアップが完了しました: \$DB_BACKUP_FILE"
    # データベースバックアップを圧縮
    gzip "\$DB_BACKUP_FILE"
    echo "データベースバックアップを圧縮しました: \$DB_BACKUP_FILE.gz"
else
    echo "データベースのバックアップに失敗しました"
    rm -f "\$DB_BACKUP_FILE"
fi

# アップロードファイルのバックアップ
echo "アップロードファイルをバックアップ中..."
UPLOADS_BACKUP_FILE="\$BACKUP_DIR/uploads_backup_\$TIMESTAMP.tar.gz"
tar -czf "\$UPLOADS_BACKUP_FILE" -C "\$APP_DIR" uploads/

# 古いバックアップを削除（7日以上前のもの）
echo "古いバックアップを削除中..."
find "\$BACKUP_DIR" -name "db_backup_*.sql.gz" -type f -mtime +7 -delete
find "\$BACKUP_DIR" -name "uploads_backup_*.tar.gz" -type f -mtime +7 -delete

echo "バックアップ処理が完了しました"
EOF
    chmod +x "$BACKUP_SCRIPT"
    
    # cronジョブの設定
    if ! crontab -l | grep -q "$BACKUP_SCRIPT"; then
        (crontab -l 2>/dev/null; echo "0 3 * * * $BACKUP_SCRIPT") | crontab -
        echo "バックアップを毎日午前3時に実行するようにcronジョブを設定しました"
    else
        echo "バックアップのcronジョブはすでに設定されています"
    fi
    echo "バックアップスクリプトを作成しました"
else
    echo "バックアップスクリプトはすでに存在します"
fi

# プロジェクトのリポジトリをクローン
print_step "10. プロジェクトファイルを取得しています..."
if [ ! -f "$APP_DIR/go.mod" ]; then
    # 現在の作業ディレクトリを確認
    if [ -f "go.mod" ] && [ "$(pwd)" != "$APP_DIR" ]; then
        echo "現在のディレクトリからファイルをコピーします"
        cp -r * "$APP_DIR/" 2>/dev/null || true
        cp -r .* "$APP_DIR/" 2>/dev/null || true
    else
        echo "GitHubからリポジトリをクローンしています..."
        cd "$APP_DIR"
        git init
        git remote add origin https://github.com/SketchShifter/sketchshifter_backend.git
        git fetch --depth=1
        git checkout main 2>/dev/null || git checkout master 2>/dev/null || echo "リポジトリからチェックアウトできませんでした。マニュアルでファイルを追加してください。"
    fi
else
    echo "プロジェクトファイルはすでに存在します"
fi

# Makefileの作成
print_step "11. Makefileを作成しています..."
if [ ! -f "$APP_DIR/Makefile" ]; then
    cat > "$APP_DIR/Makefile" << 'EOF'
.PHONY: build run stop logs migrate-up migrate-down clean help prod prod-logs

# デフォルトのターゲット
help:
	@echo "使用可能なコマンド:"
	@echo "  make prod          - 本番用アプリケーションを実行"
	@echo "  make prod-logs     - 本番用アプリケーションのログを表示"
	@echo "  make stop          - アプリケーションを停止"
	@echo "  make migrate-up    - マイグレーションを実行"
	@echo "  make migrate-down  - マイグレーションをロールバック"
	@echo "  make clean         - すべてのコンテナとボリュームを削除"

# 本番用アプリケーションを実行
prod:
	@echo "本番用アプリケーションを起動中..."
	@docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	docker-compose -f docker-compose.prod.yml up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "ログを表示するには 'make prod-logs' を実行してください。"

# 本番用アプリケーションのログを表示
prod-logs:
	@echo "本番用アプリケーションのログを表示中..."
	docker-compose -f docker-compose.prod.yml logs -f api

# アプリケーションを停止
stop:
	@echo "アプリケーションを停止中..."
	docker-compose -f docker-compose.prod.yml down
	@echo "アプリケーションを停止しました。"

# マイグレーションを実行
migrate-up:
	@echo "マイグレーションを実行中..."
	@if docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	else \
		echo "APIコンテナが実行されていません。先に 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションが完了しました！"

# マイグレーションをロールバック
migrate-down:
	@echo "マイグレーションをロールバック中..."
	@if docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate down; \
	else \
		echo "APIコンテナが実行されていません。先に 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションのロールバックが完了しました！"

# すべてのコンテナとボリュームを削除
clean:
	@echo "すべてのコンテナとボリュームを削除中..."
	docker-compose -f docker-compose.prod.yml down -v
	docker rmi -f $(docker images -q processing-api) 2>/dev/null || true
	@echo "クリーンアップが完了しました！"
EOF
    echo "Makefileを作成しました"
fi

# セットアップの完了
print_step "セットアップが完了しました！"
echo ""
echo "変更を適用するには、以下のコマンドでログアウトして再ログインしてください："
echo "  exit"
echo ""
echo "再ログイン後、アプリケーションを起動するには："
echo "  cd $APP_DIR"
echo "  make prod"
echo ""
echo "マイグレーションを実行するには："
echo "  cd $APP_DIR"
echo "  make migrate-up"
echo ""
echo "ログを確認するには："
echo "  cd $APP_DIR"
echo "  make prod-logs"
echo ""
echo "自動起動サービスを手動で開始するには："
echo "  sudo systemctl start sketchshifter.service"
echo ""
echo "手動バックアップを実行するには："
echo "  $BACKUP_SCRIPT"
echo ""
echo "重要: Dockerグループの変更を適用するにはログアウトが必要です！"