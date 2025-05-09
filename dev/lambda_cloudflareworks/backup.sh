#!/bin/bash
# backup.sh - Processing作品共有プラットフォームのバックアップスクリプト

# バックアップディレクトリの設定
BACKUP_DIR="$HOME/backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
APP_DIR="$HOME/sketchshifter_backend"

# ディレクトリが存在しない場合は作成
mkdir -p "$BACKUP_DIR"

# 環境変数の読み込み
cd "$APP_DIR"
if [ -f .env ]; then
    source .env
else
    echo "環境変数ファイル(.env)が見つかりません"
    exit 1
fi

# データベースのバックアップ
echo "データベースをバックアップ中..."
DB_BACKUP_FILE="$BACKUP_DIR/db_backup_$TIMESTAMP.sql"
docker-compose -f docker-compose.prod.yml exec -T db mysqldump \
    -u root \
    -p"${MYSQL_ROOT_PASSWORD:-root_password}" \
    "${DB_NAME:-processing_platform}" > "$DB_BACKUP_FILE"

# バックアップが正常に作成されたか確認
if [ -s "$DB_BACKUP_FILE" ]; then
    echo "データベースのバックアップが完了しました: $DB_BACKUP_FILE"
    # データベースバックアップを圧縮
    gzip "$DB_BACKUP_FILE"
    echo "データベースバックアップを圧縮しました: $DB_BACKUP_FILE.gz"
else
    echo "データベースのバックアップに失敗しました"
    rm -f "$DB_BACKUP_FILE"
fi

# アップロードファイルのバックアップ
echo "アップロードファイルをバックアップ中..."
UPLOADS_BACKUP_FILE="$BACKUP_DIR/uploads_backup_$TIMESTAMP.tar.gz"
tar -czf "$UPLOADS_BACKUP_FILE" -C "$APP_DIR" uploads/

# バックアップが正常に作成されたか確認
if [ -s "$UPLOADS_BACKUP_FILE" ]; then
    echo "アップロードファイルのバックアップが完了しました: $UPLOADS_BACKUP_FILE"
else
    echo "アップロードファイルのバックアップに失敗しました"
    rm -f "$UPLOADS_BACKUP_FILE"
fi

# 古いバックアップを削除（7日以上前のもの）
echo "古いバックアップを削除中..."
find "$BACKUP_DIR" -name "db_backup_*.sql.gz" -type f -mtime +7 -delete
find "$BACKUP_DIR" -name "uploads_backup_*.tar.gz" -type f -mtime +7 -delete

echo "バックアップ処理が完了しました"

# S3へのバックアップのアップロード (オプション、AWS CLIがインストールされている場合)
if command -v aws >/dev/null 2>&1; then
    echo "Amazon S3へのバックアップを開始します..."
    
    # S3バケット名が設定されているか確認
    S3_BUCKET="${S3_BACKUP_BUCKET:-}"
    if [ -z "$S3_BUCKET" ]; then
        echo "S3バケットが設定されていません。スキップします。"
        echo "S3へのアップロードを有効にするには、.envファイルにS3_BACKUP_BUCKETを設定してください。"
    else
        # 圧縮されたDBバックアップとアップロードバックアップをS3にアップロード
        aws s3 cp "$DB_BACKUP_FILE.gz" "s3://$S3_BUCKET/database/"
        aws s3 cp "$UPLOADS_BACKUP_FILE" "s3://$S3_BUCKET/uploads/"
        echo "Amazon S3へのバックアップが完了しました"
    fi
fi

echo "すべての処理が完了しました"