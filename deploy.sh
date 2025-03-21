#!/bin/bash
# deploy.sh - Processing作品共有プラットフォームのデプロイスクリプト

set -e

# 色の定義
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
echo "====================================================="
echo "  Processing作品共有プラットフォーム デプロイツール  "
echo "====================================================="
echo -e "${NC}"

# ディレクトリチェック
if [ ! -f "go.mod" ] || [ ! -f "Makefile" ]; then
    print_error "正しいディレクトリにいることを確認してください。go.modとMakefileが必要です。"
    exit 1
fi

# コンフィグチェック
if [ ! -f "Dockerfile.prod" ]; then
    print_warning "Dockerfile.prodが見つかりません。既存のDockerfileを複製します。"
    cp Dockerfile Dockerfile.prod
fi

if [ ! -f "docker-compose.prod.yml" ]; then
    print_warning "docker-compose.prod.ymlが見つかりません。デフォルトの設定を使用します。"
    cp docker-compose.yml docker-compose.prod.yml
    # 本番用の設定に変更
    sed -i 's/GIN_MODE=debug/GIN_MODE=release/g' docker-compose.prod.yml
fi

# 環境変数ファイルの確認
if [ ! -f ".env" ]; then
    print_step "環境変数ファイルを作成しています..."
    echo "DB_USER=processing_user" > .env
    echo "DB_PASSWORD=$(openssl rand -base64 12 | tr -dc 'a-zA-Z0-9')" >> .env
    echo "DB_NAME=processing_platform" >> .env
    echo "MYSQL_ROOT_PASSWORD=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9')" >> .env
    echo "JWT_SECRET=$(openssl rand -base64 32)" >> .env
    print_step "環境変数ファイル (.env) が作成されました。必要に応じて編集してください。"
fi

# アップロードディレクトリの作成
print_step "アップロードディレクトリを確認しています..."
mkdir -p uploads
chmod 755 uploads

# ビルドとデプロイ
print_step "アプリケーションをビルドしています..."
docker-compose -f docker-compose.prod.yml build

print_step "アプリケーションを起動しています..."
docker-compose -f docker-compose.prod.yml up -d

# マイグレーションを実行
print_step "データベースマイグレーションを実行しています..."
sleep 10  # DBが起動するまで少し待つ
docker-compose -f docker-compose.prod.yml exec api /app/app migrate up

# ヘルスチェック
print_step "ヘルスチェックを実行しています..."
sleep 5  # APIが起動するまで少し待つ

API_URL="http://localhost:8080/api/v1/health"
HEALTH_CHECK=$(curl -s -o /dev/null -w "%{http_code}" $API_URL || echo "failed")

if [ "$HEALTH_CHECK" == "200" ]; then
    print_step "アプリケーションが正常に起動しました！"
    echo ""
    echo "API URL: http://localhost:8080/api/v1"
    echo "ヘルスチェック: $API_URL"
    echo ""
    echo "ログを確認するには次のコマンドを実行してください:"
    echo "  docker-compose -f docker-compose.prod.yml logs -f api"
else
    print_error "ヘルスチェックに失敗しました。ログを確認してください:"
    docker-compose -f docker-compose.prod.yml logs api
fi

echo ""
echo -e "${GREEN}デプロイプロセスが完了しました！${NC}"