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
	@sudo docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	sudo docker-compose -f docker-compose.prod.yml up -d
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
	@if docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
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
