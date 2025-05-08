'use strict';

// AWS SDK V3のインポート
import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient, GetCommand, PutCommand, UpdateCommand } from "@aws-sdk/lib-dynamodb";

// Lambda@Edge環境では、us-east-1リージョンでDynamoDBにアクセスする必要があります
const client = new DynamoDBClient({ region: 'us-east-1' });
const dynamodb = DynamoDBDocumentClient.from(client);

// レート制限設定
const RATE_LIMITS = {
    DEFAULT: {
        requestsPerMinute: 60, // デフォルトの制限：1分間に60リクエスト
        window: 60 // 時間枠（秒）
    },
    // 特定のパスに対する制限
    '/api/v1/works': {
        requestsPerMinute: 30, // 作品APIは1分間に30リクエスト
        window: 60
    },
    '/api/v1/processing': {
        requestsPerMinute: 10, // Processing変換は1分間に10リクエスト
        window: 60
    }
};

// DynamoDBテーブル名
const TABLE_NAME = 'sketchshifter-rate-limits';

export const handler = async (event, context) => {
    const request = event.Records[0].cf.request;
    
    // IPアドレスの取得
    const clientIP = request.headers['x-forwarded-for'] && request.headers['x-forwarded-for'][0]
        ? request.headers['x-forwarded-for'][0].value.split(',')[0].trim()
        : 'unknown';
    
    // リクエストパスに基づいて適切なレート制限を取得
    const path = request.uri;
    const limit = RATE_LIMITS[path] || RATE_LIMITS.DEFAULT;
    
    try {
        // IPアドレスとパスに基づいてレート制限をチェック
        const isAllowed = await checkRateLimit(clientIP, path, limit);
        
        if (!isAllowed) {
            // レート制限を超えた場合は429エラーを返す
            return {
                status: '429',
                statusDescription: 'Too Many Requests',
                headers: {
                    'content-type': [
                        {
                            key: 'Content-Type',
                            value: 'application/json'
                        }
                    ],
                    'access-control-allow-origin': [
                        {
                            key: 'Access-Control-Allow-Origin',
                            value: '*'
                        }
                    ]
                },
                body: JSON.stringify({
                    error: 'Rate limit exceeded. Please try again later.'
                })
            };
        }
        
        // リクエストを許可
        return request;
    } catch (error) {
        console.error('Error in rate limiting:', error);
        
        // エラーが発生した場合でもリクエストを許可（フェイルオープン）
        return request;
    }
};

// レート制限チェック関数
async function checkRateLimit(clientIP, path, limit) {
    const now = Math.floor(Date.now() / 1000);
    const key = `${clientIP}:${path}`;
    const windowStart = now - limit.window;
    
    try {
        // DynamoDBからこのIPアドレスとパスの使用状況を取得
        const getCommand = new GetCommand({
            TableName: TABLE_NAME,
            Key: {
                id: key
            }
        });
        
        const response = await dynamodb.send(getCommand);
        const item = response.Item;
        
        if (!item) {
            // 新しいレコードを作成
            const putCommand = new PutCommand({
                TableName: TABLE_NAME,
                Item: {
                    id: key,
                    count: 1,
                    timestamp: now,
                    expiration: now + limit.window + 60 // TTL: window + 1分
                }
            });
            
            await dynamodb.send(putCommand);
            return true;
        }
        
        // 時間枠が経過していればカウンターをリセット
        if (item.timestamp < windowStart) {
            const updateCommand = new UpdateCommand({
                TableName: TABLE_NAME,
                Key: {
                    id: key
                },
                UpdateExpression: 'SET #count = :count, #timestamp = :timestamp, #expiration = :expiration',
                ExpressionAttributeNames: {
                    '#count': 'count',
                    '#timestamp': 'timestamp',
                    '#expiration': 'expiration'
                },
                ExpressionAttributeValues: {
                    ':count': 1,
                    ':timestamp': now,
                    ':expiration': now + limit.window + 60
                }
            });
            
            await dynamodb.send(updateCommand);
            return true;
        }
        
        // 制限に達していればfalseを返す
        if (item.count >= limit.requestsPerMinute) {
            return false;
        }
        
        // カウンターを更新
        const updateCommand = new UpdateCommand({
            TableName: TABLE_NAME,
            Key: {
                id: key
            },
            UpdateExpression: 'SET #count = #count + :incr',
            ExpressionAttributeNames: {
                '#count': 'count'
            },
            ExpressionAttributeValues: {
                ':incr': 1
            }
        });
        
        await dynamodb.send(updateCommand);
        return true;
    } catch (error) {
        console.error('DynamoDB error:', error);
        // エラー時はデフォルトで許可
        return true;
    }
}