#!/bin/bash

# 環境変数を読み込み
source ./../.env

# 閾値を設定
THRESHOLD=100

# データベースから未処理の画像数を取得
QUERY="SELECT COUNT(*) as count FROM image_processing_queue WHERE status = 'pending'"
COUNT=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME -se "$QUERY")

echo "Pending images: $COUNT"

# 閾値チェック
if [ $COUNT -ge $THRESHOLD ]; then
    echo "Threshold reached. Processing images..."
    
    # 未処理の画像を取得
    QUERY="SELECT id, file_path, original_key FROM image_processing_queue WHERE status = 'pending' LIMIT 100"
    IMAGES=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME -se "$QUERY")
    
    # ステータスを処理中に更新
    IDS=$(echo "$IMAGES" | awk '{print $1}' | tr '\n' ',' | sed 's/,$//')
    if [ ! -z "$IDS" ]; then
        UPDATE_QUERY="UPDATE image_processing_queue SET status = 'processing' WHERE id IN ($IDS)"
        mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME -e "$UPDATE_QUERY"
        
        # SQSにメッセージを送信
        for image in $IMAGES; do
            ID=$(echo $image | awk '{print $1}')
            FILE_PATH=$(echo $image | awk '{print $2}')
            ORIGINAL_KEY=$(echo $image | awk '{print $3}')
            
            # JSONメッセージを作成
            MESSAGE="{\"id\":\"$ID\",\"bucket\":\"$R2_BUCKET_NAME\",\"key\":\"$ORIGINAL_KEY\"}"
            
            # SQSにメッセージを送信
            aws sqs send-message \
                --queue-url $SQS_WEBP_QUEUE_URL \
                --message-body "$MESSAGE"
                
            echo "Sent to SQS: $MESSAGE"
        done
    fi
fi
