#!/bin/bash
# api-gateway-setup.sh - API Gateway設定自動化スクリプト

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

# AWS CLIがインストールされていることを確認
if ! command -v aws &> /dev/null; then
    print_error "AWS CLIがインストールされていません。先にインストールしてください。"
    exit 1
fi

# JQがインストールされていることを確認
if ! command -v jq &> /dev/null; then
    print_step "jqをインストールしています..."
    sudo yum install -y jq
fi

# AWSクレデンシャルを確認
if ! aws sts get-caller-identity &> /dev/null; then
    print_error "AWS CLIの認証情報が設定されていないか、無効です。"
    print_error "aws configureを実行して認証情報を設定してください。"
    exit 1
fi

# バナー表示
echo -e "${GREEN}"
echo "====================================================="
echo "  API Gateway & VPC Link セットアップツール  "
echo "====================================================="
echo -e "${NC}"

# 変数設定
APP_DIR="/home/ec2-user/ssjs/sketchshifter_backend"
AWS_REGION=$(aws configure get region)
if [ -z "$AWS_REGION" ]; then
    AWS_REGION="ap-northeast-1" # デフォルトリージョン
fi

# API名の設定
API_NAME="processing-platform-api"
STAGE_NAME="prod"

# EC2情報の取得
print_step "EC2インスタンス情報を取得しています..."
EC2_INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
EC2_PRIVATE_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4)
EC2_AZ=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
EC2_VPC_ID=$(aws ec2 describe-instances --instance-ids "$EC2_INSTANCE_ID" --region "$AWS_REGION" --query 'Reservations[0].Instances[0].VpcId' --output text)
EC2_SUBNET_ID=$(aws ec2 describe-instances --instance-ids "$EC2_INSTANCE_ID" --region "$AWS_REGION" --query 'Reservations[0].Instances[0].SubnetId' --output text)
EC2_SECURITY_GROUP_ID=$(aws ec2 describe-instances --instance-ids "$EC2_INSTANCE_ID" --region "$AWS_REGION" --query 'Reservations[0].Instances[0].SecurityGroups[0].GroupId' --output text)

echo "Instance ID: $EC2_INSTANCE_ID"
echo "Private IP: $EC2_PRIVATE_IP"
echo "VPC ID: $EC2_VPC_ID"
echo "Subnet ID: $EC2_SUBNET_ID"
echo "Security Group ID: $EC2_SECURITY_GROUP_ID"

# セキュリティグループの更新（API Gatewayからのアクセスを許可）
print_step "セキュリティグループを更新しています..."
if ! aws ec2 describe-security-groups --group-ids "$EC2_SECURITY_GROUP_ID" --region "$AWS_REGION" --query "SecurityGroups[0].IpPermissions[?FromPort==\`8080\` && ToPort==\`8080\` && IpProtocol==\`tcp\`]" --output text | grep -q 8080; then
    aws ec2 authorize-security-group-ingress \
        --group-id "$EC2_SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 8080 \
        --cidr "0.0.0.0/0" \
        --region "$AWS_REGION"
    echo "セキュリティグループにポート8080のルールを追加しました"
else
    echo "セキュリティグループには既にポート8080のルールが存在します"
fi

# NLBの作成
print_step "Network Load Balancerを作成しています..."
NLB_NAME="processing-platform-nlb"

# 既存のNLBを確認
EXISTING_NLB=$(aws elbv2 describe-load-balancers --region "$AWS_REGION" --query "LoadBalancers[?LoadBalancerName=='$NLB_NAME'].LoadBalancerArn" --output text)

if [ -z "$EXISTING_NLB" ]; then
    NLB_ARN=$(aws elbv2 create-load-balancer \
        --name "$NLB_NAME" \
        --type network \
        --scheme internal \
        --subnets "$EC2_SUBNET_ID" \
        --region "$AWS_REGION" \
        --query 'LoadBalancers[0].LoadBalancerArn' \
        --output text)
    echo "NLBを作成しました: $NLB_ARN"

    # ターゲットグループの作成
    TG_ARN=$(aws elbv2 create-target-group \
        --name "${NLB_NAME}-tg" \
        --protocol TCP \
        --port 8080 \
        --vpc-id "$EC2_VPC_ID" \
        --target-type instance \
        --region "$AWS_REGION" \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text)
    echo "ターゲットグループを作成しました: $TG_ARN"

    # EC2インスタンスをターゲットグループに追加
    aws elbv2 register-targets \
        --target-group-arn "$TG_ARN" \
        --targets Id="$EC2_INSTANCE_ID" \
        --region "$AWS_REGION"
    echo "EC2インスタンスをターゲットグループに登録しました"

    # リスナーの作成
    LISTENER_ARN=$(aws elbv2 create-listener \
        --load-balancer-arn "$NLB_ARN" \
        --protocol TCP \
        --port 8080 \
        --default-actions Type=forward,TargetGroupArn="$TG_ARN" \
        --region "$AWS_REGION" \
        --query 'Listeners[0].ListenerArn' \
        --output text)
    echo "リスナーを作成しました: $LISTENER_ARN"
else
    NLB_ARN=$EXISTING_NLB
    echo "既存のNLBを使用します: $NLB_ARN"
    
    # 既存のターゲットグループを取得
    TG_ARN=$(aws elbv2 describe-target-groups \
        --load-balancer-arn "$NLB_ARN" \
        --region "$AWS_REGION" \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text)
    
    # EC2インスタンスが登録されているか確認
    TARGET_HEALTH=$(aws elbv2 describe-target-health \
        --target-group-arn "$TG_ARN" \
        --targets Id="$EC2_INSTANCE_ID" \
        --region "$AWS_REGION" \
        --query 'TargetHealthDescriptions[0].TargetHealth.State' \
        --output text 2>/dev/null || echo "not_registered")
    
    if [ "$TARGET_HEALTH" == "not_registered" ]; then
        # EC2インスタンスをターゲットグループに追加
        aws elbv2 register-targets \
            --target-group-arn "$TG_ARN" \
            --targets Id="$EC2_INSTANCE_ID" \
            --region "$AWS_REGION"
        echo "EC2インスタンスをターゲットグループに登録しました"
    else
        echo "EC2インスタンスは既にターゲットグループに登録されています (状態: $TARGET_HEALTH)"
    fi
fi

# VPCリンクの作成
print_step "VPC Linkを作成しています..."
VPC_LINK_NAME="processing-platform-vpc-link"

# 既存のVPCリンクを確認
EXISTING_VPC_LINK_ID=$(aws apigateway get-vpc-links \
    --region "$AWS_REGION" \
    --query "items[?name==\`$VPC_LINK_NAME\`].id" \
    --output text)

if [ -z "$EXISTING_VPC_LINK_ID" ]; then
    VPC_LINK_ID=$(aws apigateway create-vpc-link \
        --name "$VPC_LINK_NAME" \
        --target-arns "$NLB_ARN" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "VPC Linkを作成しました: $VPC_LINK_ID"
    
    # VPCリンクが有効になるまで待機
    print_step "VPC Linkが有効になるまで待機しています..."
    aws apigateway get-vpc-link \
        --vpc-link-id "$VPC_LINK_ID" \
        --region "$AWS_REGION" \
        --query 'status' \
        --output text

    # 最大5分間待機
    for i in {1..30}; do
        VPC_LINK_STATUS=$(aws apigateway get-vpc-link \
            --vpc-link-id "$VPC_LINK_ID" \
            --region "$AWS_REGION" \
            --query 'status' \
            --output text)
        
        echo "VPC Link状態: $VPC_LINK_STATUS"
        
        if [ "$VPC_LINK_STATUS" == "AVAILABLE" ]; then
            break
        fi
        
        if [ "$i" -eq 30 ]; then
            print_warning "VPC Linkの準備に時間がかかっています。スクリプトを継続しますが、APIの設定が完了しない可能性があります。"
        fi
        
        sleep 10
    done
else
    VPC_LINK_ID=$EXISTING_VPC_LINK_ID
    echo "既存のVPC Linkを使用します: $VPC_LINK_ID"
fi

# API Gatewayの作成
print_step "API Gatewayを作成しています..."

# 既存のAPIを確認
EXISTING_API_ID=$(aws apigateway get-rest-apis \
    --region "$AWS_REGION" \
    --query "items[?name==\`$API_NAME\`].id" \
    --output text)

if [ -z "$EXISTING_API_ID" ]; then
    API_ID=$(aws apigateway create-rest-api \
        --name "$API_NAME" \
        --description "Processing Platform API" \
        --endpoint-configuration "{ \"types\": [\"REGIONAL\"] }" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "APIを作成しました: $API_ID"
else
    API_ID=$EXISTING_API_ID
    echo "既存のAPIを使用します: $API_ID"
fi

# ルートリソースのIDを取得
ROOT_RESOURCE_ID=$(aws apigateway get-resources \
    --rest-api-id "$API_ID" \
    --region "$AWS_REGION" \
    --query 'items[?path==`/`].id' \
    --output text)

# /api リソースを作成または取得
API_RESOURCE_PATH="/api"
API_RESOURCE_ID=$(aws apigateway get-resources \
    --rest-api-id "$API_ID" \
    --region "$AWS_REGION" \
    --query "items[?path==\`$API_RESOURCE_PATH\`].id" \
    --output text)

if [ -z "$API_RESOURCE_ID" ]; then
    API_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$ROOT_RESOURCE_ID" \
        --path-part "api" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "/api リソースを作成しました: $API_RESOURCE_ID"
else
    echo "既存の /api リソースを使用します: $API_RESOURCE_ID"
fi

# /api/v1 リソースを作成または取得
V1_RESOURCE_PATH="/api/v1"
V1_RESOURCE_ID=$(aws apigateway get-resources \
    --rest-api-id "$API_ID" \
    --region "$AWS_REGION" \
    --query "items[?path==\`$V1_RESOURCE_PATH\`].id" \
    --output text)

if [ -z "$V1_RESOURCE_ID" ]; then
    V1_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$API_RESOURCE_ID" \
        --path-part "v1" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "/api/v1 リソースを作成しました: $V1_RESOURCE_ID"
else
    echo "既存の /api/v1 リソースを使用します: $V1_RESOURCE_ID"
fi

# /api/v1/{proxy+} リソースを作成または取得
PROXY_RESOURCE_PATH="/api/v1/{proxy+}"
PROXY_RESOURCE_ID=$(aws apigateway get-resources \
    --rest-api-id "$API_ID" \
    --region "$AWS_REGION" \
    --query "items[?path==\`$PROXY_RESOURCE_PATH\`].id" \
    --output text)

if [ -z "$PROXY_RESOURCE_ID" ]; then
    PROXY_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$V1_RESOURCE_ID" \
        --path-part "{proxy+}" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "/api/v1/{proxy+} リソースを作成しました: $PROXY_RESOURCE_ID"
else
    echo "既存の /api/v1/{proxy+} リソースを使用します: $PROXY_RESOURCE_ID"
fi

# ANY メソッドをプロキシリソースに設定
print_step "プロキシリソースにANYメソッドを設定しています..."

# 統合リクエスト用のJSON生成
INTEGRATION_JSON=$(cat <<EOF
{
  "type": "HTTP_PROXY",
  "httpMethod": "ANY",
  "uri": "http://$EC2_PRIVATE_IP:8080/api/v1/{proxy}",
  "connectionType": "VPC_LINK",
  "connectionId": "$VPC_LINK_ID",
  "requestParameters": {
    "integration.request.path.proxy": "method.request.path.proxy"
  }
}
EOF
)

# メソッドリクエスト用のJSON生成
METHOD_JSON=$(cat <<EOF
{
  "authorizationType": "NONE",
  "apiKeyRequired": false,
  "requestParameters": {
    "method.request.path.proxy": true
  }
}
EOF
)

# メソッドが存在するか確認
METHOD_EXISTS=$(aws apigateway get-method \
    --rest-api-id "$API_ID" \
    --resource-id "$PROXY_RESOURCE_ID" \
    --http-method "ANY" \
    --region "$AWS_REGION" 2>/dev/null || echo "NOT_EXISTS")

if [ "$METHOD_EXISTS" == "NOT_EXISTS" ]; then
    # メソッドを作成
    aws apigateway put-method \
        --rest-api-id "$API_ID" \
        --resource-id "$PROXY_RESOURCE_ID" \
        --http-method "ANY" \
        --authorization-type "NONE" \
        --request-parameters "{ \"method.request.path.proxy\": true }" \
        --region "$AWS_REGION"
    
    # 統合を設定
    aws apigateway put-integration \
        --rest-api-id "$API_ID" \
        --resource-id "$PROXY_RESOURCE_ID" \
        --http-method "ANY" \
        --type "HTTP_PROXY" \
        --integration-http-method "ANY" \
        --uri "http://$EC2_PRIVATE_IP:8080/api/v1/{proxy}" \
        --connection-type "VPC_LINK" \
        --connection-id "$VPC_LINK_ID" \
        --request-parameters "{ \"integration.request.path.proxy\": \"method.request.path.proxy\" }" \
        --region "$AWS_REGION"
    
    echo "ANYメソッドと統合を作成しました"
else
    # 統合を更新
    aws apigateway update-integration \
        --rest-api-id "$API_ID" \
        --resource-id "$PROXY_RESOURCE_ID" \
        --http-method "ANY" \
        --patch-operations "[{\"op\":\"replace\",\"path\":\"/uri\",\"value\":\"http://$EC2_PRIVATE_IP:8080/api/v1/{proxy}\"},{\"op\":\"replace\",\"path\":\"/connectionId\",\"value\":\"$VPC_LINK_ID\"}]" \
        --region "$AWS_REGION"
    
    echo "ANYメソッドの統合を更新しました"
fi

# ヘルスチェックメソッドを設定
print_step "ヘルスチェックメソッドを設定しています..."

# /api/v1/health リソースを作成または取得
HEALTH_RESOURCE_PATH="/api/v1/health"
HEALTH_RESOURCE_ID=$(aws apigateway get-resources \
    --rest-api-id "$API_ID" \
    --region "$AWS_REGION" \
    --query "items[?path==\`$HEALTH_RESOURCE_PATH\`].id" \
    --output text)

if [ -z "$HEALTH_RESOURCE_ID" ]; then
    HEALTH_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$V1_RESOURCE_ID" \
        --path-part "health" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    echo "/api/v1/health リソースを作成しました: $HEALTH_RESOURCE_ID"
else
    echo "既存の /api/v1/health リソースを使用します: $HEALTH_RESOURCE_ID"
fi

# GETメソッドが存在するか確認
HEALTH_METHOD_EXISTS=$(aws apigateway get-method \
    --rest-api-id "$API_ID" \
    --resource-id "$HEALTH_RESOURCE_ID" \
    --http-method "GET" \
    --region "$AWS_REGION" 2>/dev/null || echo "NOT_EXISTS")

if [ "$HEALTH_METHOD_EXISTS" == "NOT_EXISTS" ]; then
    # GETメソッドを作成
    aws apigateway put-method \
        --rest-api-id "$API_ID" \
        --resource-id "$HEALTH_RESOURCE_ID" \
        --http-method "GET" \
        --authorization-type "NONE" \
        --region "$AWS_REGION"
    
    # 統合を設定
    aws apigateway put-integration \
        --rest-api-id "$API_ID" \
        --resource-id "$HEALTH_RESOURCE_ID" \
        --http-method "GET" \
        --type "HTTP_PROXY" \
        --integration-http-method "GET" \
        --uri "http://$EC2_PRIVATE_IP:8080/api/v1/health" \
        --connection-type "VPC_LINK" \
        --connection-id "$VPC_LINK_ID" \
        --region "$AWS_REGION"
    
    echo "ヘルスチェックのGETメソッドと統合を作成しました"
else
    # 統合を更新
    aws apigateway update-integration \
        --rest-api-id "$API_ID" \
        --resource-id "$HEALTH_RESOURCE_ID" \
        --http-method "GET" \
        --patch-operations "[{\"op\":\"replace\",\"path\":\"/uri\",\"value\":\"http://$EC2_PRIVATE_IP:8080/api/v1/health\"},{\"op\":\"replace\",\"path\":\"/connectionId\",\"value\":\"$VPC_LINK_ID\"}]" \
        --region "$AWS_REGION"
    
    echo "ヘルスチェックのGETメソッドの統合を更新しました"
fi

# APIをデプロイ
print_step "APIをデプロイしています..."
DEPLOYMENT_ID=$(aws apigateway create-deployment \
    --rest-api-id "$API_ID" \
    --stage-name "$STAGE_NAME" \
    --description "Production deployment" \
    --region "$AWS_REGION" \
    --query 'id' \
    --output text)

echo "APIをデプロイしました: $DEPLOYMENT_ID"

# バイナリメディアタイプを設定
print_step "バイナリメディアタイプを設定しています..."
aws apigateway update-rest-api \
    --rest-api-id "$API_ID" \
    --patch-operations "[{\"op\":\"add\",\"path\":\"/binaryMediaTypes/*~1*\",\"value\":\"\"}]" \
    --region "$AWS_REGION"

echo "バイナリメディアタイプを設定しました"

# レート制限の設定
print_step "レート制限を設定しています..."
USAGE_PLAN_NAME="processing-platform-usage-plan"

# 既存の使用量プランを確認
EXISTING_USAGE_PLAN_ID=$(aws apigateway get-usage-plans \
    --region "$AWS_REGION" \
    --query "items[?name==\`$USAGE_PLAN_NAME\`].id" \
    --output text)

if [ -z "$EXISTING_USAGE_PLAN_ID" ]; then
    USAGE_PLAN_ID=$(aws apigateway create-usage-plan \
        --name "$USAGE_PLAN_NAME" \
        --description "Rate limits for Processing Platform API" \
        --throttle "rateLimit=10, burstLimit=20" \
        --quota "limit=1000, period=DAY" \
        --region "$AWS_REGION" \
        --query 'id' \
        --output text)
    
    # 使用量プランにステージを追加
    aws apigateway update-usage-plan \
        --usage-plan-id "$USAGE_PLAN_ID" \
        --patch-operations "[{\"op\":\"add\",\"path\":\"/apiStages\",\"value\":\"$API_ID:$STAGE_NAME\"}]" \
        --region "$AWS_REGION"
    
    echo "使用量プランを作成しました: $USAGE_PLAN_ID"
else
    USAGE_PLAN_ID=$EXISTING_USAGE_PLAN_ID
    
    # 使用量プランを更新
    aws apigateway update-usage-plan \
        --usage-plan-id "$USAGE_PLAN_ID" \
        --patch-operations "[{\"op\":\"replace\",\"path\":\"/throttle/rateLimit\",\"value\":\"10\"},{\"op\":\"replace\",\"path\":\"/throttle/burstLimit\",\"value\":\"20\"}]" \
        --region "$AWS_REGION"
    
    # 既存のステージを確認
    STAGE_EXISTS=$(aws apigateway get-usage-plan \
        --usage-plan-id "$USAGE_PLAN_ID" \
        --region "$AWS_REGION" \
        --query "apiStages[?apiId==\`$API_ID\` && stage==\`$STAGE_NAME\`]" \
        --output text)
    
    if [ -z "$STAGE_EXISTS" ]; then
        # 使用量プランにステージを追加
        aws apigateway update-usage-plan \
            --usage-plan-id "$USAGE_PLAN_ID" \
            --patch-operations "[{\"op\":\"add\",\"path\":\"/apiStages\",\"value\":\"$API_ID:$STAGE_NAME\"}]" \
            --region "$AWS_REGION"
    fi
    
    echo "既存の使用量プランを更新しました: $USAGE_PLAN_ID"
fi

# APIキーの作成（オプション）
read -p "APIキーを作成しますか？ (y/n): " CREATE_API_KEY
if [[ "$CREATE_API_KEY" =~ ^[Yy]$ ]]; then
    print_step "APIキーを作成しています..."
    API_KEY_NAME="processing-platform-api-key"
    
    # 既存のAPIキーを確認
    EXISTING_API_KEY_ID=$(aws apigateway get-api-keys \
        --name-query "$API_KEY_NAME" \
        --include-values \
        --region "$AWS_REGION" \
        --query "items[0].id" \
        --output text)
    
    if [ -z "$EXISTING_API_KEY_ID" ]; then
        API_KEY_ID=$(aws apigateway create-api-key \
            --name "$API_KEY_NAME" \
            --description "API Key for Processing Platform" \
            --enabled \
            --region "$AWS_REGION" \
            --query 'id' \
            --output text)
        
        # APIキーを使用量プランに関連付け
        aws apigateway create-usage-plan-key \
            --usage-plan-id "$USAGE_PLAN_ID" \
            --key-id "$API_KEY_ID" \
            --key-type "API_KEY" \
            --region "$AWS_REGION"
        
        # APIキーの値を取得
        API_KEY_VALUE=$(aws apigateway get-api-key \
            --api-key "$API_KEY_ID" \
            --include-value \
            --region "$AWS_REGION" \
            --query 'value' \
            --output text)
        
        echo "APIキーを作成しました: $API_KEY_ID"
        echo "APIキー値: $API_KEY_VALUE"
        echo "APIキーを使用量プランに関連付けました"
    else
        API_KEY_ID=$EXISTING_API_KEY_ID
        
        # APIキーの値を取得
        API_KEY_VALUE=$(aws apigateway get-api-key \
            --api-key "$API_KEY_ID" \
            --include-value \
            --region "$AWS_REGION" \
            --query 'value' \
            --output text)
        
        echo "既存のAPIキーを使用します: $API_KEY_ID"
        echo "APIキー値: $API_KEY_VALUE"
    fi
fi

# APIのURLを表示
API_URL="https://${API_ID}.execute-api.${AWS_REGION}.amazonaws.com/${STAGE_NAME}"
echo -e "\n${GREEN}API Gateway設定が完了しました！${NC}"
echo "API URL: $API_URL"
echo "ヘルスチェックエンドポイント: $API_URL/api/v1/health"

# Cloudfrontの設定手順を表示
echo -e "\n${YELLOW}次のステップ:${NC}"
echo "1. CloudFrontで新しいディストリビューションを作成し、オリジンとして上記のAPI URLを設定します。"
echo "2. CloudflareのDNS設定で、api.yourdomain.comを新しいCloudFrontディストリビューションにポイントします。"
echo "3. Lambda@EdgeとDynamoDBを使用したレート制限機能を設定します。"
echo ""
echo "以上の設定が完了したら、最後に古いローカルデータベースコンテナとボリュームを削除します:"
echo "docker volume rm processing_db_data"
echo ""
echo "詳細は「RDS移行ガイド」ドキュメントを参照してください。"