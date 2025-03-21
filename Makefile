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
	@sudo docker network inspect processing_network >/dev/null 2>&1 || sudo docker network create processing_network
	sudo docker-compose -f docker-compose.prod.yml up -d
	@echo "アプリケーションが起動しました。http://localhost:8080 にアクセスしてください。"
	@echo "ログを表示するには 'make prod-logs' を実行してください。"

# 本番用アプリケーションのログを表示
prod-logs:
	@echo "本番用アプリケーションのログを表示中..."
	sudo docker-compose -f docker-compose.prod.yml logs -f api

# アプリケーションを停止
stop:
	@echo "アプリケーションを停止中..."
	sudo docker-compose -f docker-compose.prod.yml down
	@echo "アプリケーションを停止しました。"

# マイグレーションを実行
migrate-up:
	@echo "マイグレーションを実行中..."
	@if sudo docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		sudo docker-compose -f docker-compose.prod.yml exec api /app/app migrate up; \
	else \
		echo "APIコンテナが実行されていません。先に 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションが完了しました！"

# マイグレーションをロールバック
migrate-down:
	@echo "マイグレーションをロールバック中..."
	@if sudo docker-compose -f docker-compose.prod.yml ps | grep -q "processing-api"; then \
		sudo docker-compose -f docker-compose.prod.yml exec api /app/app migrate down; \
	else \
		echo "APIコンテナが実行されていません。先に 'make prod' を実行してください。"; \
		exit 1; \
	fi
	@echo "マイグレーションのロールバックが完了しました！"

# すべてのコンテナとボリュームを削除
clean:
	@echo "すべてのコンテナとボリュームを削除中..."
	sudo docker-compose -f docker-compose.prod.yml down -v
	sudo docker rmi -f $$(docker images -q processing-api) 2>/dev/null || true
	@echo "クリーンアップが完了しました！"