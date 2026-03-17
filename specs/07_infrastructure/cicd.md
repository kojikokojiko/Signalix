# CI/CD パイプライン仕様

## 概要

GitHub Actions を使用した CI/CD。
`main` ブランチへのマージが本番デプロイを自動でトリガーする。

---

## ブランチ戦略

```
main          ← 本番環境（直接プッシュ禁止）
staging       ← ステージング環境（main へのマージ前に検証）
feature/*     ← 機能開発ブランチ
fix/*         ← バグ修正ブランチ
```

**PR フロー:**
1. `feature/*` → `main` へ PR。
2. CI（テスト・lint・ビルド）が全て通過すること。
3. コードレビュー 1 名以上の承認。
4. マージ → 自動デプロイ。

---

## GitHub Actions ワークフロー

### 1. CI ワークフロー (`ci.yml`)

PR 作成・更新時に実行。

```yaml
name: CI

on:
  pull_request:
    branches: [main, staging]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: pgvector/pgvector:pg16
        env:
          POSTGRES_DB: signalix_test
          POSTGRES_USER: signalix
          POSTGRES_PASSWORD: test_password
        ports: ["5432:5432"]
        options: --health-cmd pg_isready --health-interval 10s
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
      - name: Run migrations
        run: make migrate-test
      - name: Run tests
        run: make test
        env:
          TEST_DB_URL: postgres://signalix:test_password@localhost:5432/signalix_test
          TEST_REDIS_URL: redis://localhost:6379
      - name: Upload coverage
        uses: codecov/codecov-action@v4

  backend-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
        working-directory: ./frontend
      - run: npm run type-check
        working-directory: ./frontend
      - run: npm run lint
        working-directory: ./frontend
      - run: npm run test
        working-directory: ./frontend

  build-check:
    runs-on: ubuntu-latest
    needs: [backend-test, backend-lint, frontend-test]
    steps:
      - uses: actions/checkout@v4
      - name: Build backend Docker image
        run: docker build -t signalix-api ./backend
      - name: Build frontend Docker image
        run: docker build -t signalix-frontend ./frontend
      - name: Build worker Docker image
        run: docker build -t signalix-worker ./worker
```

---

### 2. CD ワークフロー (`deploy.yml`)

`main` ブランチへのプッシュ時に実行。

```yaml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production
    permissions:
      id-token: write  # OIDC 認証
      contents: read
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials (OIDC)
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::${{ secrets.AWS_ACCOUNT_ID }}:role/signalix-github-deploy
          aws-region: us-east-1

      - name: Login to ECR
        uses: aws-actions/amazon-ecr-login@v2

      - name: Build and push API image
        env:
          ECR_REGISTRY: ${{ steps.login-ecr.outputs.registry }}
          IMAGE_TAG: ${{ github.sha }}
        run: |
          docker build -t $ECR_REGISTRY/signalix-api:$IMAGE_TAG ./backend
          docker push $ECR_REGISTRY/signalix-api:$IMAGE_TAG
          docker tag $ECR_REGISTRY/signalix-api:$IMAGE_TAG $ECR_REGISTRY/signalix-api:latest
          docker push $ECR_REGISTRY/signalix-api:latest

      - name: Build and push other images
        # (frontend, worker も同様)

      - name: Run DB migrations
        run: |
          aws ecs run-task \
            --cluster signalix-production \
            --task-definition signalix-migrate \
            --overrides '{"containerOverrides": [{"name": "migrate", "command": ["migrate", "up"]}]}'

      - name: Deploy API to ECS
        run: |
          aws ecs update-service \
            --cluster signalix-production \
            --service signalix-api \
            --force-new-deployment

      - name: Deploy Frontend to ECS
        run: |
          aws ecs update-service \
            --cluster signalix-production \
            --service signalix-frontend \
            --force-new-deployment

      - name: Wait for stability
        run: |
          aws ecs wait services-stable \
            --cluster signalix-production \
            --services signalix-api signalix-frontend

      - name: Notify deployment
        if: always()
        # Slack 通知（オプション）
```

---

### 3. ステージングデプロイ (`deploy-staging.yml`)

`staging` ブランチへのプッシュ時に実行。内容は `deploy.yml` とほぼ同じ。
環境変数で `staging` クラスターを指定。

---

## Makefile コマンド定義

```makefile
# backend/Makefile

.PHONY: test test-integration test-coverage lint build migrate-up migrate-down

# ユニット + 統合テスト
test:
    go test ./... -v -race -timeout 60s

# 統合テストのみ
test-integration:
    go test ./... -v -tags integration -timeout 120s

# カバレッジレポート生成
test-coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html

# リント
lint:
    golangci-lint run ./...

# ビルド
build:
    go build -o bin/api ./cmd/api
    go build -o bin/worker ./cmd/worker

# マイグレーション
migrate-up:
    migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
    migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-test:
    migrate -path ./migrations -database "$(TEST_DB_URL)" up
```

---

## デプロイ安全性チェック

デプロイ前に自動実行するチェックリスト:

1. **マイグレーション安全性**: 追加のみ（既存カラムの削除・変更は Blue/Green デプロイ後）。
2. **ヘルスチェック待機**: ECS サービス安定化を `ecs wait` で確認してから完了とする。
3. **ロールバック手順**: ECS サービスの "previous task definition" に即時ロールバック可能。
4. **DB マイグレーション**: アプリデプロイ前に実行（後方互換性を保つ）。

---

## コンテナイメージ設計

### API サーバー Dockerfile

```dockerfile
# マルチステージビルド
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM gcr.io/distroless/static-debian12
COPY --from=builder /api /api
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
```

### フロントエンド Dockerfile

```dockerfile
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --only=production

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV production
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
USER nextjs
EXPOSE 3000
CMD ["node_modules/.bin/next", "start"]
```
