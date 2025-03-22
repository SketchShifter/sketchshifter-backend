#!/bin/bash
# processing-platform-setup.sh
# Processing作品共有プラットフォームのローカル環境セットアップスクリプト
# Windows（Git Bash）、macOS、Linuxで動作します

set -e

# カラー定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# OS検出
detect_os() {
  if [ "$(uname)" == "Darwin" ]; then
    echo "macOS"
  elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    echo "Linux"
  elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW32_NT" ] || [ "$(expr substr $(uname -s) 1 10)" == "MINGW64_NT" ]; then
    echo "Windows"
  else
    echo "Unknown"
  fi
}

OS_TYPE=$(detect_os)

# ヘルパー関数
print_step() {
  echo -e "${GREEN}==>${NC} $1"
}

print_info() {
  echo -e "${BLUE}INFO:${NC} $1"
}

print_warning() {
  echo -e "${YELLOW}警告:${NC} $1"
}

print_error() {
  echo -e "${RED}エラー:${NC} $1"
}

confirm() {
  read -p "$1 (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    return 1
  fi
  return 0
}

# バナー表示
echo -e "${GREEN}"
echo "======================================================="
echo "  Processing作品共有プラットフォーム ローカルセットアップ  "
echo "======================================================="
echo -e "${NC}"

print_info "検出されたOS: $OS_TYPE"

# 作業ディレクトリの設定
DEFAULT_DIR="./processing-platform"
print_step "作業ディレクトリを設定します"
read -p "インストール先ディレクトリを入力してください [$DEFAULT_DIR]: " INSTALL_DIR
INSTALL_DIR=${INSTALL_DIR:-$DEFAULT_DIR}

# Dockerのチェック
print_step "Dockerのインストール状況をチェックしています..."
if ! command -v docker &> /dev/null; then
  print_error "Dockerが見つかりません。先にDockerをインストールしてください。"
  echo "インストール方法: https://docs.docker.com/get-docker/"
  exit 1
else
  DOCKER_VERSION=$(docker --version)
  print_info "Docker: $DOCKER_VERSION"
fi

# Docker Composeのチェック
if ! command -v docker-compose &> /dev/null; then
  print_error "Docker Composeが見つかりません。先にDocker Composeをインストールしてください。"
  echo "インストール方法: https://docs.docker.com/compose/install/"
  exit 1
else
  COMPOSE_VERSION=$(docker-compose --version)
  print_info "Docker Compose: $COMPOSE_VERSION"
fi

# Dockerデーモンが起動しているか確認
print_step "Dockerサービスの状態を確認しています..."
if ! docker info &> /dev/null; then
  print_error "Dockerデーモンが実行されていません。Dockerサービスを起動してください。"
  
  case $OS_TYPE in
    "Linux")
      echo "実行するコマンド: sudo systemctl start docker"
      ;;
    "macOS")
      echo "Docker Desktopを起動してください。"
      ;;
    "Windows")
      echo "Docker Desktopを起動してください。"
      ;;
  esac
  
  exit 1
fi

# ディレクトリ作成
print_step "プロジェクトディレクトリを作成しています..."
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"
INSTALL_DIR=$(pwd)  # 絶対パスに変換
print_info "インストール先ディレクトリ: $INSTALL_DIR"

# 設定ファイルの作成
print_step "設定ファイルを作成しています..."

# docker-compose.yml
print_info "docker-compose.ymlを作成しています..."
cat > docker-compose.yml << 'EOF'
services:
  api:
    build: .
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
      - GIN_MODE=debug
    volumes:
      - ./uploads:/app/uploads
      - .:/app
    depends_on:
      - db
    networks:
      - processing_network

  db:
    image: mariadb:10.6
    container_name: processing-db
    restart: always
    ports:
      - "3307:3306"
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-root_password}
      - MYSQL_DATABASE=${DB_NAME:-processing_platform}
      - MYSQL_USER=${DB_USER:-processing_user}
      - MYSQL_PASSWORD=${DB_PASSWORD:-processing_password}
    volumes:
      - db_data:/var/lib/mysql
    networks:
      - processing_network

  # PHPMyAdminを追加
  phpmyadmin:
    image: phpmyadmin/phpmyadmin
    container_name: phpmyadmin
    ports:
      - "8081:80"
    environment:
      - PMA_HOST=db
      - PMA_USER=${DB_USER:-processing_user}
      - PMA_PASSWORD=${DB_PASSWORD:-processing_password}
    depends_on:
      - db
    networks:
      - processing_network

networks:
  processing_network:
    driver: bridge

volumes:
  db_data:
EOF

# docker-compose.prod.yml
print_info "docker-compose.prod.ymlを作成しています..."
cat > docker-compose.prod.yml << 'EOF'
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
      interval: 300s
      timeout: 10s
      retries: 3
      start_period: 15s

  db:
    image: mariadb:10.6
    container_name: processing-db
    restart: always
    ports:
      - "127.0.0.1:3307:3306"
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
      interval: 100s
      timeout: 5s
      retries: 3
      start_period: 30s

networks:
  processing_network:
    driver: bridge

volumes:
  db_data:
EOF

# Dockerfile
print_info "Dockerfileを作成しています..."
cat > Dockerfile << 'EOF'
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
EOF

# Dockerfile.prod
print_info "Dockerfile.prodを作成しています..."
cat > Dockerfile.prod << 'EOF'
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

# docker-entrypoint.sh
print_info "docker-entrypoint.shを作成しています..."
cat > docker-entrypoint.sh << 'EOF'
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
chmod +x docker-entrypoint.sh

# Makefile
print_info "Makefileを作成しています..."
case $OS_TYPE in
  "Windows")
    # Windows用のMakefileはタブ文字を正しく処理するために別の方法で作成
    cat > Makefile.tmp << EOF
.PHONY: build run dev stop logs migrate-up migrate-down clean help prod prod-logs

# デフォルトのターゲット
help:
	@echo "使用可能なコマンド:"
	@echo "  make build      - Dockerイメージをビルド"
	@echo "  make run        - 開発用アプリケーションを実行"
	@echo "  make dev        - 開発モードでファイル変更を監視（ホットリロード）"
	@echo "  make prod       - 本番用アプリケーションを実行"
	@echo "  make stop       - アプリケーションを停止"
	@echo "  make logs       - 開発用アプリケーションのログを表示"
	@echo "  make prod-logs  - 本番用アプリケーションのログを表示"
	@echo "  make migrate-up - マイグレーションを実行"
	@echo "  make migrate-down - マイグレーションをロールバック"
	@echo "  make clean      - すべてのコンテナとボリュームを削除"

# Dockerイメージをビルド
build:
	@echo "Dockerイメージをビルド中..."
	docker-compose build

# 開発用アプリケーションを実行
run:
	@echo "開発用アプリケーションを起動中..."
	@sudo docker network inspect processing_network >/dev/null 2>&1 || sudo docker network create processing_network
	sudo docker-compose up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "PHPMyAdminは http://localhost:8081 で利用可能です。"
	@echo "ログを表示するには 'make logs' を実行してください。"

# 開発モードで実行（ホットリロード）
dev:
	@echo "開発モード（ホットリロード）でアプリケーションを起動中..."
	@sudo docker network inspect processing_network >/dev/null 2>&1 || sudo docker network create processing_network
	sudo docker-compose up
	
# 本番用アプリケーションを実行
prod:
	@echo "本番用アプリケーションを起動中..."
	@sudo docker network inspect processing_network >/dev/null 2>&1 || sudo docker network create processing_network
	sudo docker-compose -f docker-compose.prod.yml up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "ログを表示するには 'make prod-logs' を実行してください。"

# アプリケーションを停止
stop:
	@echo "アプリケーションを停止中..."
	docker-compose down
	docker-compose -f docker-compose.prod.yml down
	@echo "アプリケーションを停止しました。"

# 開発用アプリケーションのログを表示
logs:
	@echo "開発用アプリケーションのログを表示中..."
	docker-compose logs -f api

# 本番用アプリケーションのログを表示
prod-logs:
	@echo "本番用アプリケーションのログを表示中..."
	docker-compose -f docker-compose.prod.yml logs -f api

# マイグレーションを実行
migrate-up:
	@echo "マイグレーションを実行中..."
	@if docker-compose ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	elif docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	else \
		echo "APIコンテナが実行されていません。先に 'make run' または 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションが完了しました！"

# マイグレーションをロールバック
migrate-down:
	@echo "マイグレーションをロールバック中..."
	@if docker-compose ps | grep -q "processing-api"; then \
		docker-compose exec api go run cmd/migrate/main.go down; \
	elif docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate down; \
	else \
		echo "APIコンテナが実行されていません。先に 'make run' または 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションのロールバックが完了しました！"

# すべてのコンテナとボリュームを削除
clean:
	@echo "すべてのコンテナとボリュームを削除中..."
	docker-compose down -v
	docker-compose -f docker-compose.prod.yml down -v
	docker rmi -f \$$(docker images -q processing-api) 2>/dev/null || true
	@echo "クリーンアップが完了しました！"
EOF
    # タブ文字を保持したままファイルを書き出す（Windowsでの改行コードの問題を回避）
    awk '{ sub(/^[ \t]+/, "\t"); print }' Makefile.tmp > Makefile
    rm Makefile.tmp
    ;;
  *)
    # macOSとLinux用
    cat > Makefile << 'EOF'
.PHONY: build run dev stop logs migrate-up migrate-down clean help prod prod-logs

# デフォルトのターゲット
help:
	@echo "使用可能なコマンド:"
	@echo "  make build      - Dockerイメージをビルド"
	@echo "  make run        - 開発用アプリケーションを実行"
	@echo "  make dev        - 開発モードでファイル変更を監視（ホットリロード）"
	@echo "  make prod       - 本番用アプリケーションを実行"
	@echo "  make stop       - アプリケーションを停止"
	@echo "  make logs       - 開発用アプリケーションのログを表示"
	@echo "  make prod-logs  - 本番用アプリケーションのログを表示"
	@echo "  make migrate-up - マイグレーションを実行"
	@echo "  make migrate-down - マイグレーションをロールバック"
	@echo "  make clean      - すべてのコンテナとボリュームを削除"

# Dockerイメージをビルド
build:
	@echo "Dockerイメージをビルド中..."
	docker-compose build

# 開発用アプリケーションを実行
run:
	@echo "開発用アプリケーションを起動中..."
	@docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	docker-compose up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "PHPMyAdminは http://localhost:8081 で利用可能です。"
	@echo "ログを表示するには 'make logs' を実行してください。"

# 開発モードで実行（ホットリロード）
dev:
	@echo "開発モード（ホットリロード）でアプリケーションを起動中..."
	@docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	docker-compose up
	
# 本番用アプリケーションを実行
prod:
	@echo "本番用アプリケーションを起動中..."
	@docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	docker-compose -f docker-compose.prod.yml up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "ログを表示するには 'make prod-logs' を実行してください。"

# アプリケーションを停止
stop:
	@echo "アプリケーションを停止中..."
	docker-compose down
	docker-compose -f docker-compose.prod.yml down
	@echo "アプリケーションを停止しました。"

# 開発用アプリケーションのログを表示
logs:
	@echo "開発用アプリケーションのログを表示中..."
	docker-compose logs -f api

# 本番用アプリケーションのログを表示
prod-logs:
	@echo "本番用アプリケーションのログを表示中..."
	docker-compose -f docker-compose.prod.yml logs -f api

# マイグレーションを実行
migrate-up:
	@echo "マイグレーションを実行中..."
	@if docker-compose ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	elif docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	else \
		echo "APIコンテナが実行されていません。先に 'make run' または 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションが完了しました！"

# マイグレーションをロールバック
migrate-down:
	@echo "マイグレーションをロールバック中..."
	@if docker-compose ps | grep -q "processing-api"; then \
		docker-compose exec api go run cmd/migrate/main.go down; \
	elif docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		docker-compose -f docker-compose.prod.yml exec api /app/app migrate down; \
	else \
		echo "APIコンテナが実行されていません。先に 'make run' または 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションのロールバックが完了しました！"

# すべてのコンテナとボリュームを削除
clean:
	@echo "すべてのコンテナとボリュームを削除中..."
	docker-compose down -v
	docker-compose -f docker-compose.prod.yml down -v
	docker rmi -f $$(docker images -q processing-api) 2>/dev/null || true
	@echo "クリーンアップが完了しました！"
EOF
    ;;
esac

# .env ファイルの作成
print_info ".envファイルを作成しています..."
cat > .env << EOF
DB_USER=processing_user
DB_PASSWORD=processing_password
DB_NAME=processing_platform
MYSQL_ROOT_PASSWORD=root_password
JWT_SECRET=$(LC_ALL=C tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 32 | head -n 1)
EOF

# アップロードディレクトリの作成
print_info "アップロードディレクトリを作成しています..."
mkdir -p uploads
chmod 755 uploads



# 実行コマンドをREADMEに保存
print_step "READMEファイルを作成しています..."
cat > README.md << EOF
# Processing作品共有プラットフォーム

## セットアップ方法

このプロジェクトは Docker と Docker Compose を使用して実行されます。

### 必要条件

- Docker
- Docker Compose
- make (オプション、便利なコマンドが使えます)

### インストールと実行方法

1. 開発モードで実行:

\`\`\`bash
make run
\`\`\`

2. 本番モードで実行:

\`\`\`bash
make prod
\`\`\`

3. マイグレーションを実行:

\`\`\`bash
make migrate-up
\`\`\`

4. ログを表示:

\`\`\`bash
make logs      # 開発モード
make prod-logs # 本番モード
\`\`\`

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

\`\`\`
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
\`\`\`
EOF

print_step "セットアップが完了しました！"
echo ""
echo "プロジェクトディレクトリ: $INSTALL_DIR"
echo ""
echo "開発環境を起動するには:"
echo "  cd $INSTALL_DIR"
echo "  make run"
echo ""
echo "マイグレーションを実行するには:"
echo "  make migrate-up"
echo ""
echo "本番環境を起動するには:"
echo "  make prod"
echo ""
echo "詳細は README.md を参照してください。"