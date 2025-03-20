.PHONY: build run stop logs migrate-up migrate-down clean help

# デフォルトのターゲット
help:
	@echo "使用可能なコマンド:"
	@echo "  make build      - Dockerイメージをビルド"
	@echo "  make run        - アプリケーションを実行"
	@echo "  make stop       - アプリケーションを停止"
	@echo "  make logs       - アプリケーションのログを表示"
	@echo "  make migrate-up - マイグレーションを実行"
	@echo "  make migrate-down - マイグレーションをロールバック"
	@echo "  make clean      - すべてのコンテナとボリュームを削除"

# Dockerイメージをビルド
build:
	@echo "Dockerイメージをビルド中..."
	docker compose build
	@echo "ビルド完了！"

# アプリケーションを実行
run:
	@echo "アプリケーションを起動中..."
	@docker network inspect processing_network >/dev/null 2>&1 || docker network create processing_network
	docker compose up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "PHPMyAdminは http://localhost:8081 で利用可能です。"
	@echo "ログを表示するには 'make logs' を実行してください。"

# アプリケーションを停止
stop:
	@echo "アプリケーションを停止中..."
	docker compose down
	@echo "アプリケーションを停止しました。"

# ログを表示
logs:
	@echo "アプリケーションのログを表示中..."
	docker compose logs -f api

# マイグレーションを実行
migrate-up:
	@echo "データベースコンテナが起動するまで待機中..."
	@docker compose up -d db
	@sleep 5
	@echo "マイグレーションを実行中..."
	docker compose exec api go run cmd/migrate/main.go up
	@echo "マイグレーションが完了しました！"

# マイグレーションをロールバック
migrate-down:
	@echo "データベースコンテナが起動するまで待機中..."
	@docker compose up -d db
	@sleep 5
	@echo "マイグレーションをロールバックしています..."
	docker compose exec api go run cmd/migrate/main.go down
	@echo "マイグレーションのロールバックが完了しました！"

# すべてのコンテナとボリュームを削除
clean:
	@echo "すべてのコンテナとボリュームを削除中..."
	docker compose down -v
	docker rmi -f $(shell docker compose images -q)
	@echo "クリーンアップが完了しました！"