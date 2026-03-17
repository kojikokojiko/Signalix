# 可観測性仕様

## 概要

MVP では AWS CloudWatch を中心とした可観測性基盤を構築する。
将来的に Grafana + Prometheus への移行も考慮した設計とする。

---

## ロギング

### 構造化ログ形式

全コンポーネントで JSON 構造化ログを使用する（`zap` ライブラリ）。

```json
{
  "level": "info",
  "ts": "2024-03-15T10:00:00.000Z",
  "caller": "handler/articles.go:42",
  "msg": "request completed",
  "service": "api-server",
  "version": "1.2.3",
  "environment": "production",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/v1/recommendations",
  "status_code": 200,
  "duration_ms": 45,
  "user_id": "user-uuid"
}
```

### 必須フィールド

| フィールド | 説明 |
|-----------|------|
| `level` | `debug`, `info`, `warn`, `error` |
| `ts` | ISO 8601 タイムスタンプ |
| `service` | サービス名（`api-server`, `rss-worker`, `processing-worker`, `recommendation-worker`） |
| `version` | アプリバージョン（git commit sha 7桁） |
| `environment` | `production`, `staging` |
| `msg` | ログメッセージ |

### リクエストログの必須フィールド

| フィールド | 説明 |
|-----------|------|
| `request_id` | `X-Request-ID` または自動生成 UUID |
| `method` | HTTP メソッド |
| `path` | リクエストパス |
| `status_code` | レスポンスステータスコード |
| `duration_ms` | レスポンス時間（ミリ秒） |
| `user_id` | 認証済みユーザーの ID（なければ省略） |

### ログレベルガイドライン

| レベル | 使用場面 |
|-------|---------|
| `debug` | 開発時のみ。本番では出力しない |
| `info` | 正常系の処理完了（リクエスト完了、ジョブ完了） |
| `warn` | 異常だが継続可能（リトライ発生、レートリミット近接） |
| `error` | 処理失敗（エラースタックトレースを含める） |

### CloudWatch Logs グループ

```
/signalix/production/api-server
/signalix/production/rss-worker
/signalix/production/processing-worker
/signalix/production/recommendation-worker
/signalix/production/frontend

ログ保持期間: 30 日
```

---

## メトリクス

### CloudWatch カスタムメトリクス

ECS タスクから定期的（1 分間隔）に発行するメトリクス。

#### API サーバー

| メトリクス名 | 単位 | 説明 |
|-----------|------|------|
| `signalix/api/request_count` | Count | リクエスト数（Dimension: endpoint, status_code） |
| `signalix/api/request_duration_p50` | Milliseconds | レスポンスタイム中央値 |
| `signalix/api/request_duration_p95` | Milliseconds | レスポンスタイム 95 パーセンタイル |
| `signalix/api/request_duration_p99` | Milliseconds | レスポンスタイム 99 パーセンタイル |
| `signalix/api/error_rate` | Percent | 5xx エラー率 |
| `signalix/api/active_connections` | Count | アクティブ接続数 |

#### インジェスションワーカー

| メトリクス名 | 単位 | 説明 |
|-----------|------|------|
| `signalix/ingestion/articles_new` | Count | 新規インジェスト記事数（1分間） |
| `signalix/ingestion/job_duration_ms` | Milliseconds | ジョブ実行時間 |
| `signalix/ingestion/job_failures` | Count | ジョブ失敗数 |
| `signalix/ingestion/sources_degraded` | Count | degraded 状態のソース数 |

#### 処理ワーカー

| メトリクス名 | 単位 | 説明 |
|-----------|------|------|
| `signalix/processing/articles_processed` | Count | 処理完了記事数（1分間） |
| `signalix/processing/article_duration_ms` | Milliseconds | 記事処理時間 |
| `signalix/processing/queue_depth` | Count | 処理待ちキューの深さ |
| `signalix/processing/llm_tokens_used` | Count | LLM トークン使用数 |
| `signalix/processing/stage_failures` | Count | ステージ失敗数（Dimension: stage） |

#### レコメンドワーカー

| メトリクス名 | 単位 | 説明 |
|-----------|------|------|
| `signalix/recommendation/users_refreshed` | Count | リフレッシュしたユーザー数 |
| `signalix/recommendation/duration_ms` | Milliseconds | 全ユーザー計算時間 |
| `signalix/recommendation/per_user_duration_ms` | Milliseconds | ユーザーあたりの計算時間 |

---

## アラート

### 重要度: Critical（即時対応が必要）

| アラート名 | 条件 | 通知先 |
|-----------|------|-------|
| API エラー率急上昇 | 5xx エラー率 > 5%（5 分間） | PagerDuty + Slack |
| API レスポンスタイム高 | p99 > 3000ms（5 分間） | Slack |
| RDS 接続エラー | DB 接続失敗が続く | PagerDuty + Slack |
| ECS タスク停止 | desired_count != running_count（5 分継続） | Slack |

### 重要度: Warning（確認が必要）

| アラート名 | 条件 | 通知先 |
|-----------|------|-------|
| インジェスション失敗増加 | ジョブ失敗数 > 5/時間 | Slack |
| 処理キュー深さ高 | queue_depth > 100（15 分継続） | Slack |
| ソース degraded 増加 | degraded ソース数 > 3 | Slack |
| LLM API コスト超過 | 1 日のトークン使用量が予算の 80% 超 | Slack |
| Redis メモリ使用率高 | > 80% | Slack |
| RDS CPU 高 | > 80%（10 分継続） | Slack |

### CloudWatch Alarms 設定例

```hcl
resource "aws_cloudwatch_metric_alarm" "api_error_rate" {
  alarm_name          = "signalix-api-error-rate-critical"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "signalix/api/error_rate"
  namespace           = "Signalix"
  period              = 300  # 5 分
  threshold           = 5.0
  alarm_description   = "API 5xx error rate exceeded 5%"
  alarm_actions       = [aws_sns_topic.critical.arn]
  ok_actions          = [aws_sns_topic.critical.arn]
}
```

---

## ヘルスチェック

### API サーバーヘルスチェック

```
GET /health
認証不要

レスポンス 200:
{
  "status": "ok",
  "version": "1.2.3",
  "checks": {
    "database": "ok",
    "redis": "ok"
  },
  "timestamp": "2024-03-15T10:00:00Z"
}

レスポンス 503（いずれかのチェックが失敗）:
{
  "status": "degraded",
  "checks": {
    "database": "error: connection refused",
    "redis": "ok"
  }
}
```

**ALB ヘルスチェック設定:**
- パス: `/health`
- 正常レスポンス: 200
- チェック間隔: 30 秒
- 異常閾値: 3 回連続失敗でタスク置き換え

---

## ダッシュボード

### CloudWatch ダッシュボード構成

**メインダッシュボード: "Signalix - Overview"**

```
行 1: KPI
  [新規記事/時間] [処理済み記事数] [アクティブユーザー] [LLM コスト/日]

行 2: API パフォーマンス
  [リクエスト数] [エラー率] [レスポンスタイム p50/p95/p99]

行 3: パイプライン状態
  [インジェスションジョブ成功/失敗] [処理キュー深さ] [処理ワーカー利用率]

行 4: インフラ
  [RDS CPU/IOPS/接続数] [Redis メモリ/接続数] [ECS CPU/メモリ]

行 5: エラー
  [エラーログ（直近 1 時間）] [失敗ジョブ一覧]
```

---

## 分散トレーシング（将来計画）

MVP では実装しないが、将来的な追加を考慮して以下を準備する:

- `X-Request-ID` ヘッダーを全コンポーネントで伝播する（MVP で実装済み）。
- ログに `request_id` を含めることで、ログレベルでのリクエスト追跡を可能にする。
- Phase 2 以降: AWS X-Ray または OpenTelemetry の導入を検討。
