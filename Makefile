.PHONY: up down logs migrate ps build

# ─────────────────────────────────────────
# 開発環境
# ─────────────────────────────────────────

# 全サービス起動
up:
	docker compose -f docker-compose.dev.yml up

# バックグラウンドで起動
up-d:
	docker compose -f docker-compose.dev.yml up -d

# 停止
down:
	docker compose -f docker-compose.dev.yml down

# 停止 + ボリューム削除 (DB を完全リセット)
down-v:
	docker compose -f docker-compose.dev.yml down -v

# ログ確認
logs:
	docker compose -f docker-compose.dev.yml logs -f

# 特定サービスのログ (例: make logs-api)
logs-%:
	docker compose -f docker-compose.dev.yml logs -f $*

# サービス状態確認
ps:
	docker compose -f docker-compose.dev.yml ps

# イメージ再ビルド
build:
	docker compose -f docker-compose.dev.yml build

# ─────────────────────────────────────────
# マイグレーション
# ─────────────────────────────────────────

# マイグレーション実行 (up)
migrate-up:
	docker compose -f docker-compose.dev.yml run --rm migrate

# マイグレーション 1 つ戻す (down)
migrate-down:
	docker compose -f docker-compose.dev.yml run --rm migrate \
		-path=/migrations \
		-database="postgres://$${POSTGRES_USER:-signalix}:$${POSTGRES_PASSWORD:-dev_password}@postgres:5432/$${POSTGRES_DB:-signalix_dev}?sslmode=disable" \
		down 1

# ─────────────────────────────────────────
# テスト
# ─────────────────────────────────────────

# バックエンドテスト (コンテナ内で実行)
test:
	docker compose -f docker-compose.dev.yml exec api go test ./... -v -race

# ─────────────────────────────────────────
# DB 操作
# ─────────────────────────────────────────

# psql 接続
psql:
	docker compose -f docker-compose.dev.yml exec postgres \
		psql -U $${POSTGRES_USER:-signalix} -d $${POSTGRES_DB:-signalix_dev}
